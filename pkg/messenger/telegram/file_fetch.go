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
	// FileSize is the byte count to pass to
	// StorageRepository.UploadFile's objectSize. It's -1 (unknown) when
	// neither the download response nor the Bot API could tell us a
	// trustworthy size -- callers must NOT treat 0 as "empty file" and
	// must forward -1 as-is (minio-go switches to a streaming multipart
	// upload for -1, so an unknown size is safe, just slightly less
	// efficient than a known one).
	FileSize int64
}

// FetchFile resolves a Telegram file_id to its bytes in two steps, per
// the Bot API: getFile to learn the file's server-side path, then a
// plain HTTPS GET of the resulting download link. This is the only place
// in the codebase that talks to Telegram's file-download endpoint (as
// opposed to the Bot API proper) -- keeping it here, alongside SendMessage
// and Listen, is what keeps all Telegram-specific mechanics out of the
// application/handler layers.
func FetchFile(ctx context.Context, bot *tgbot.Bot, platformFileID string) (*FetchedFile, error) {
	file, err := bot.GetFile(ctx, &tgbot.GetFileParams{FileID: platformFileID})
	if err != nil {
		return nil, fmt.Errorf("telegram: get file %q: %w", platformFileID, err)
	}

	downloadURL := bot.FileDownloadLink(file)

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

	// ? FUCK BALE : Bale fill filesizes with zero values :/
	size := resp.ContentLength
	if size <= 0 {
		size = file.FileSize
	}
	if size <= 0 {
		size = -1
	}

	return &FetchedFile{Body: resp.Body, FileSize: size}, nil
}
