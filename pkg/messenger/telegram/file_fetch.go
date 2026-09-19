package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"

	tgbot "github.com/go-telegram/bot"
)

// FetchedFile is the result of resolving a Telegram file_id
// (Attachment.PlatformFileID) to its actual bytes.
type FetchedFile struct {
	// Body is the file's content stream. Callers MUST Close() it.
	Body io.ReadCloser
	// FileSize is Telegram's own reported size for the file, in bytes
	// (0 if Telegram didn't report one -- this happens for some file
	// types). Pass it to StorageRepository.UploadFile's objectSize.
	FileSize int64
}

// FetchFile resolves a Telegram file_id to its bytes in two steps, per
// the Bot API: getFile to learn the file's server-side path, then a
// plain HTTPS GET of the resulting download link. This is the only place
// in the codebase that talks to Telegram's file-download endpoint (as
// opposed to the Bot API proper) -- keeping it here, alongside SendMessage
// and Listen, is what keeps all Telegram-specific mechanics out of the
// application/handler layers.
func (a *Adapter) FetchFile(ctx context.Context, platformFileID string) (*FetchedFile, error) {
	file, err := a.bot.GetFile(ctx, &tgbot.GetFileParams{FileID: platformFileID})
	if err != nil {
		return nil, fmt.Errorf("telegram: get file %q: %w", platformFileID, err)
	}

	downloadURL := a.bot.FileDownloadLink(file)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram: build download request for %q: %w", platformFileID, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: download file %q: %w", platformFileID, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("telegram: download file %q: unexpected status %d", platformFileID, resp.StatusCode)
	}

	return &FetchedFile{Body: resp.Body, FileSize: file.FileSize}, nil
}
