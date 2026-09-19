package telegram

// Attachment is this package's adapter-level representation of a single
// piece of media found on a Telegram message
// (photo/video/document/audio/voice/animation).
//
// It intentionally mirrors messenger-backend/internal/application/contract's
// CreateAttachmentRequest field-for-field, but package telegram does not
// import that contract package -- an adapter package has no business
// depending on an application-layer request shape. Callers (e.g. package
// telegramhandlers) map Attachment into whatever their own layer needs;
// see telegramhandlers/attachment_mapping.go for that mapping.
type Attachment struct {
	PlatformFileID          string
	ThumbnailPlatformFileID string
	FileType                string
	FileName                string
	MimeType                string
	FileSize                int64
	Width                   int
	Height                  int
	Duration                int
}

// ExtractAttachments inspects a message for media
// (photo/video/document/audio/voice/animation), returning one Attachment
// per file found -- a Telegram message carries at most one of these,
// except Photo, which arrives as several resolutions of the same image:
// the largest (by pixel area) becomes the attachment and the smallest
// becomes its ThumbnailPlatformFileID. Field names here are checked
// directly against github.com/go-telegram/bot@v1.25.0's models package,
// not inferred.
//
// Moved here from package telegramhandlers (was
// extractTelegramAttachments), unchanged apart from returning Attachment
// instead of service_contract.CreateAttachmentRequest -- this is exactly
// the kind of raw-model-shape logic that belongs behind the telegram
// adapter rather than in a handler.
func ExtractAttachments(msg *Message) (attachments []Attachment, caption string) {
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
		att := Attachment{
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
		att := Attachment{
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
		att := Attachment{
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
		attachments = append(attachments, Attachment{
			PlatformFileID: msg.Audio.FileID,
			FileType:       "audio",
			FileName:       msg.Audio.FileName,
			MimeType:       msg.Audio.MimeType,
			FileSize:       int64(msg.Audio.FileSize),
			Duration:       msg.Audio.Duration,
		})

	case msg.Voice != nil:
		attachments = append(attachments, Attachment{
			PlatformFileID: msg.Voice.FileID,
			FileType:       "voice",
			MimeType:       msg.Voice.MimeType,
			FileSize:       int64(msg.Voice.FileSize),
			Duration:       msg.Voice.Duration,
		})

	case msg.Animation != nil:
		att := Attachment{
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

// MediaTypeFromAttachments decides a ChatHistory-style MediaType from an
// extracted attachment set: "text" for none, the single FileType for
// exactly one, "mixed" for more than one (see ChatHistory.MediaType's doc
// comment on the application side for the full vocabulary this draws
// from).
//
// Moved here from package telegramhandlers (was the unexported
// telegramMediaType(attachments []service_contract.CreateAttachmentRequest)),
// unchanged apart from operating on []Attachment instead.
func MediaTypeFromAttachments(attachments []Attachment) string {
	switch len(attachments) {
	case 0:
		return "text"
	case 1:
		return attachments[0].FileType
	default:
		return "mixed"
	}
}
