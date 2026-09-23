package service

import (
	"context"
	"fmt"
	"os"
	"time"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/pkg/logger"
	mail "messenger-backend/pkg/wneessen-go-mail"
	"messenger-backend/static"

	"github.com/panjf2000/ants/v2"
)

// TODO use mq :
const emailDebugFile = "./SEND_EMAIL.txt"

type emailService struct {
	sender   mail.Sender
	fileOut  bool
	realSend bool
	statics  *static.StaticFiles
	log      logger.Logger
	pool     *ants.Pool
}

func NewEmailService(
	sender mail.Sender,
	fileOut bool,
	realSend bool,
	statics *static.StaticFiles,
	log logger.Logger,
	pool *ants.Pool,
) service_contract.EmailService {
	return &emailService{
		sender:   sender,
		fileOut:  fileOut,
		realSend: realSend,
		statics:  statics,
		log:      log.With(logger.String("component", "email_service")),
		pool:     pool,
	}
}

func (s *emailService) SendEmail(ctx context.Context, req service_contract.SendEmailRequest) (*bool, error) {
	success := true

	if s.fileOut {
		if _, err := s.writeEmailToFile(req); err != nil {
			s.log.Error(err, "failed to write email to debug file",
				logger.Any("to", req.To),
				logger.String("subject", req.Subject),
			)
		}
	}

	if s.realSend {
		err := s.pool.Submit(func() {
			if err := s.sender.Send(context.Background(), mail.SendOptions{
				To:          req.To,
				Subject:     req.Subject,
				PlainBody:   req.TextBody,
				HTMLBody:    req.HTMLBody,
				EmbedFiles:  req.EmbedFiles,
				Attachments: req.Attachments,
			}); err != nil {
				s.log.Error(err, "failed to send email via SMTP provider",
					logger.Any("to", req.To),
					logger.String("subject", req.Subject),
				)
			} else {
				s.log.Info("email sent successfully via SMTP provider",
					logger.Any("to", req.To),
					logger.String("subject", req.Subject),
				)
			}
		})

		if err != nil {
			s.log.Error(err, "email pool is full or closed, dropping email", logger.Any("to", req.To))
			return nil, fmt.Errorf("email pool capacity exceeded: %w", err)
		}
	}

	return &success, nil
}

func (s *emailService) writeEmailToFile(req service_contract.SendEmailRequest) (*bool, error) {
	success := true

	debugText := fmt.Sprintf(
		"[%s] To: %s | Subject: %s\nText: %s\n-------------------------------------------------------------------------------\n",
		time.Now().Format(time.RFC3339), req.To, req.Subject, req.TextBody,
	)

	emailFile, err := os.OpenFile(emailDebugFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		s.log.Error(err, "failed to open email debug file", logger.Any("to", req.To))
		return nil, fmt.Errorf("failed to open email debug log file: %w", err)
	}
	defer emailFile.Close()

	if _, err := emailFile.WriteString(debugText); err != nil {
		s.log.Error(err, "failed to write to email debug file", logger.Any("to", req.To))
		return nil, fmt.Errorf("failed to write to email debug log file: %w", err)
	}

	return &success, nil
}

func (s *emailService) SendOTPEmail(ctx context.Context, req service_contract.SendOTPEmailRequest) (*bool, error) {
	buffer, err := s.statics.ExecuteOTPTemplate(req.OTP, req.TTL)
	if err != nil {
		s.log.Error(err, "failed to execute OTP HTML template", logger.Any("to", req.To))
		return nil, fmt.Errorf("failed to render OTP email template: %w", err)
	}

	plainTextBody := fmt.Sprintf("Your OTP code is: %s", req.OTP)

	success, err := s.SendEmail(ctx, service_contract.SendEmailRequest{
		To:       req.To,
		Subject:  "OTP-Email-Verification-For-BMM",
		HTMLBody: buffer.String(),
		TextBody: plainTextBody,
	})
	if err != nil {
		return nil, err
	}

	return success, nil
}
