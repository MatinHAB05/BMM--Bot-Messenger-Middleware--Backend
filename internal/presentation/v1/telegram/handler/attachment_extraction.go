package telegramhandlers

import (
	service_contract "messenger-backend/internal/application/contract"

	"github.com/go-telegram/bot/models"
)

// extractTelegramMessage picks the message-bearing field off whichever
// update type arrived (a normal message, an edit, or a channel post all
// carry the same shape).
func extractTelegramMessage(update *models.Update) *models.Message {
	switch {
	case update.Message != nil:
		return update.Message
	case update.EditedMessage != nil:
		return update.EditedMessage
	case update.ChannelPost != nil:
		return update.ChannelPost
	default:
		return nil
	}
}

// extractTelegramAttachments inspects a message for media
// (photo/video/document/audio/voice/animation), returning one
// CreateAttachmentRequest per file found -- a Telegram message carries at
// most one of these, except Photo, which arrives as several resolutions
// of the same image: the largest (by pixel area) becomes the attachment
// and the smallest becomes its ThumbnailPlatformFileID. Field names here
// are checked directly against github.com/go-telegram/bot@v1.25.0's
// models package, not inferred.
func extractTelegramAttachments(msg *models.Message) (attachments []service_contract.CreateAttachmentRequest, caption string) {
	switch {
	case len(msg.Photo) > 0:
		largest, smallest := msg.Photo[0], msg.Photo[0]
		for _, p := range msg.Photo {
			if p.Width*p.Height > largest.Width*largest.Height {
				largest = p
			}
			if p.Width*p.Height < smallest.Width*smallest.Height {
				smallest = p
			}
		}
		att := service_contract.CreateAttachmentRequest{
			PlatformFileID: largest.FileID,
			FileType:       "photo",
			FileSize:       int64(largest.FileSize),
			Width:          largest.Width,
			Height:         largest.Height,
		}
		if smallest.FileID != largest.FileID {
			att.ThumbnailPlatformFileID = smallest.FileID
		}
		attachments = append(attachments, att)

	case msg.Document != nil:
		att := service_contract.CreateAttachmentRequest{
			PlatformFileID: msg.Document.FileID,
			FileType:       "document",
			FileName:       msg.Document.FileName,
			MimeType:       msg.Document.MimeType,
			FileSize:       int64(msg.Document.FileSize),
		}
		if msg.Document.Thumbnail != nil {
			att.ThumbnailPlatformFileID = msg.Document.Thumbnail.FileID
		}
		attachments = append(attachments, att)

	case msg.Video != nil:
		att := service_contract.CreateAttachmentRequest{
			PlatformFileID: msg.Video.FileID,
			FileType:       "video",
			MimeType:       msg.Video.MimeType,
			FileSize:       int64(msg.Video.FileSize),
			Width:          msg.Video.Width,
			Height:         msg.Video.Height,
			Duration:       msg.Video.Duration,
		}
		if msg.Video.Thumbnail != nil {
			att.ThumbnailPlatformFileID = msg.Video.Thumbnail.FileID
		}
		attachments = append(attachments, att)

	case msg.Audio != nil:
		attachments = append(attachments, service_contract.CreateAttachmentRequest{
			PlatformFileID: msg.Audio.FileID,
			FileType:       "audio",
			FileName:       msg.Audio.FileName,
			MimeType:       msg.Audio.MimeType,
			FileSize:       int64(msg.Audio.FileSize),
			Duration:       msg.Audio.Duration,
		})

	case msg.Voice != nil:
		attachments = append(attachments, service_contract.CreateAttachmentRequest{
			PlatformFileID: msg.Voice.FileID,
			FileType:       "voice",
			MimeType:       msg.Voice.MimeType,
			FileSize:       int64(msg.Voice.FileSize),
			Duration:       msg.Voice.Duration,
		})

	case msg.Animation != nil:
		att := service_contract.CreateAttachmentRequest{
			PlatformFileID: msg.Animation.FileID,
			FileType:       "animation",
			FileName:       msg.Animation.FileName,
			MimeType:       msg.Animation.MimeType,
			FileSize:       int64(msg.Animation.FileSize),
			Width:          msg.Animation.Width,
			Height:         msg.Animation.Height,
			Duration:       msg.Animation.Duration,
		}
		if msg.Animation.Thumbnail != nil {
			att.ThumbnailPlatformFileID = msg.Animation.Thumbnail.FileID
		}
		attachments = append(attachments, att)
	}

	return attachments, msg.Caption
}

// telegramMediaType decides ChatHistory.MediaType from the extracted
// attachment set: "text" for none, the single FileType for exactly one,
// "mixed" for more than one (see ChatHistory.MediaType's doc comment for
// the full vocabulary this draws from).
func telegramMediaType(attachments []service_contract.CreateAttachmentRequest) string {
	switch len(attachments) {
	case 0:
		return "text"
	case 1:
		return attachments[0].FileType
	default:
		return "mixed"
	}
}
