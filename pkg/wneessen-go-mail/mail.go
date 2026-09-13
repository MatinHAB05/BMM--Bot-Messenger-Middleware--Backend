package mail

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wneessen/go-mail"
)

type EmailConfig struct {
	From     string
	Password string
	SMTPHost string
	SMTPPort string
}

type SendOptions struct {
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	PlainBody   string
	HTMLBody    string
	EmbedFiles  []string
	Attachments []string
}

type Sender interface {
	Send(ctx context.Context, opts SendOptions) error
}

type mailer struct {
	cfg EmailConfig
}

func NewMailer(cfg EmailConfig) Sender {
	return &mailer{cfg: cfg}
}

func (m *mailer) Send(ctx context.Context, opts SendOptions) error {
	message := mail.NewMsg()

	if err := message.From(m.cfg.From); err != nil {
		return fmt.Errorf("failed to set sender email: %w", err)
	}

	if err := message.To(opts.To...); err != nil {
		return fmt.Errorf("failed to set recipient emails: %w", err)
	}

	if len(opts.Cc) > 0 {
		if err := message.Cc(opts.Cc...); err != nil {
			return fmt.Errorf("failed to set CC emails: %w", err)
		}
	}

	if len(opts.Bcc) > 0 {
		if err := message.Bcc(opts.Bcc...); err != nil {
			return fmt.Errorf("failed to set BCC emails: %w", err)
		}
	}

	message.Subject(opts.Subject)

	for _, file := range opts.EmbedFiles {
		if file != "" {
			message.EmbedFile(file)
		}
	}

	if opts.PlainBody != "" {
		message.SetBodyString(mail.TypeTextPlain, opts.PlainBody)
	}

	if opts.HTMLBody != "" {
		message.AddAlternativeString(mail.TypeTextHTML, opts.HTMLBody)
	}

	for _, attachment := range opts.Attachments {
		if attachment != "" {
			message.AttachFile(attachment)
		}
	}

	port, err := strconv.Atoi(m.cfg.SMTPPort)
	if err != nil {
		return fmt.Errorf("invalid SMTP port '%s': %w", m.cfg.SMTPPort, err)
	}

	client, err := mail.NewClient(
		m.cfg.SMTPHost,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(m.cfg.From),
		mail.WithPassword(m.cfg.Password),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
