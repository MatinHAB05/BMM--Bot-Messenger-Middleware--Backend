// Package bale is a self-contained adapter for the Bale (بله) messenger's
// Bot API (https://tapi.bale.ai/bot<token>/<method>). Although Bale's Bot
// API is conceptually modeled after Telegram's, this package deliberately
// implements its own HTTP client and payload types rather than reusing
// pkg/messenger/telegram -- the two engines are fully decoupled and share
// no code, only the pkg/messenger.MessengerClient contract.
package simplebale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://tapi.bale.ai/bot"

// ChatUpdate is the minimal, platform-neutral shape of information the
// background listener forwards for persistence -- see the identical
// rationale in pkg/messenger/telegram.ChatUpdate. Duplicated intentionally
// rather than shared, to keep the two engines independently compilable.
type ChatUpdate struct {
	TargetID string
	Title    string
	ChatType string
}

// UpdateHandler is invoked for every chat observed via an incoming update.
type UpdateHandler func(ctx context.Context, update ChatUpdate)

// Adapter implements messenger.MessengerClient for Bale.
type Adapter struct {
	token              string
	httpClient         *http.Client
	onUpdate           UpdateHandler
	pollTimeoutSeconds int
}

// NewAdapter constructs a Bale adapter around the given bot token.
// onUpdate may be nil if only outbound SendMessage capability is needed.
func NewAdapter(token string, onUpdate UpdateHandler) *Adapter {
	return &Adapter{
		token:              token,
		httpClient:         &http.Client{Timeout: 35 * time.Second},
		onUpdate:           onUpdate,
		pollTimeoutSeconds: 30,
	}
}

func (a *Adapter) Platform() string {
	return "bale"
}

func (a *Adapter) endpoint(method string) string {
	return fmt.Sprintf("%s%s/%s", baseURL, a.token, method)
}

type apiEnvelope struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
}

// post serializes payload as JSON, POSTs it to the given Bale Bot API
// method, and returns the raw "result" field once the envelope's "ok" flag
// has been checked.
func (a *Adapter) post(ctx context.Context, method string, payload interface{}) (json.RawMessage, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint(method), bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}

	if !envelope.OK {
		return nil, fmt.Errorf("bale api error (status %d, code %d): %s", resp.StatusCode, envelope.ErrorCode, envelope.Description)
	}

	return envelope.Result, nil
}

type sendMessageRequest struct {
	ChatID interface{} `json:"chat_id"`
	Text   string      `json:"text"`
}

func (a *Adapter) SendMessage(ctx context.Context, targetID string, content string) error {
	_, err := a.post(ctx, "sendMessage", sendMessageRequest{ChatID: chatID(targetID), Text: content})
	if err != nil {
		return fmt.Errorf("bale: send message to %q failed: %w", targetID, err)
	}
	return nil
}

type getUpdatesRequest struct {
	Offset  int64 `json:"offset,omitempty"`
	Timeout int   `json:"timeout"`
}

type incomingUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID        int64  `json:"id"`
			Type      string `json:"type"`
			Title     string `json:"title"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
		} `json:"chat"`
	} `json:"message"`
}

// Listen starts long-polling Bale's getUpdates endpoint. It blocks until
// ctx is cancelled, so callers should run it in its own goroutine (see
// bootstrap/init.go). onFailure, if non-nil, is called with transient
// polling errors (the loop backs off and retries rather than exiting).
func (a *Adapter) Listen(ctx context.Context, onFailure func(err error)) {
	var offset int64

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := a.post(ctx, "getUpdates", getUpdatesRequest{Offset: offset, Timeout: a.pollTimeoutSeconds})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if onFailure != nil {
				onFailure(err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}

		var updates []incomingUpdate
		if err := json.Unmarshal(result, &updates); err != nil {
			if onFailure != nil {
				onFailure(fmt.Errorf("decode updates: %w", err))
			}
			continue
		}

		for _, u := range updates {
			offset = u.UpdateID + 1

			if u.Message == nil || a.onUpdate == nil {
				continue
			}

			chat := u.Message.Chat
			title := chat.Title
			if title == "" {
				title = strings.TrimSpace(chat.FirstName + " " + chat.LastName)
			}
			if title == "" {
				title = chat.Username
			}

			a.onUpdate(ctx, ChatUpdate{
				TargetID: strconv.FormatInt(chat.ID, 10),
				Title:    title,
				ChatType: chat.Type,
			})
		}
	}
}

// chatID lets a target be specified either as a numeric Bale chat id or an
// @username -- both are accepted by the Bale Bot API's chat_id field.
func chatID(targetID string) interface{} {
	if id, err := strconv.ParseInt(targetID, 10, 64); err == nil {
		return id
	}
	return targetID
}
