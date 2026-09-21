package balehandlers

import (
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/pkg/messenger/telegram"
)

// toCreateAttachmentRequests maps the adapter-level telegram.Attachment
// set (produced by telegram.ExtractAttachments) into this service's own
// CreateAttachmentRequest shape. Straight field-for-field copy, in the
// same order telegram.ExtractAttachments populates them -- no
// Telegram-specific knowledge lives here anymore, all of that (field
// names, which struct fields apply to which media type, the
// largest/smallest photo selection, etc.) is now in package telegram.
// This is the one place that's allowed to know about both shapes, since
// it's the boundary between the adapter and the application layer.
func toCreateAttachmentRequests(atts []telegram.Attachment) []service_contract.CreateAttachmentRequest {
	if len(atts) == 0 {
		return nil
	}

	out := make([]service_contract.CreateAttachmentRequest, len(atts))
	for i, a := range atts {
		out[i] = service_contract.CreateAttachmentRequest{
			PlatformFileID:          a.PlatformFileID,
			ThumbnailPlatformFileID: a.ThumbnailPlatformFileID,
			FileType:                a.FileType,
			FileName:                a.FileName,
			MimeType:                a.MimeType,
			FileSize:                a.FileSize,
			Width:                   a.Width,
			Height:                  a.Height,
			Duration:                a.Duration,
		}
	}
	return out
}
