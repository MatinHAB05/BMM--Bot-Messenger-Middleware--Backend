package balemiddleware

import (
	"context"
	"fmt"
	"strings"

	"messenger-backend/internal/domain/exception"
	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger/telegram"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var errLogger logger.Logger

func InitErrLogger(l logger.Logger) {
	errLogger = l
}

type (
	linkOTPCtxKey             struct{}
	deepStartGroupOTPCtxKey   struct{}
	deepStartChannelOTPCtxKey struct{}
)

// CommandParser defines the interface for command parsing strategy.
type CommandParser interface {
	Parse(text string) (string, error)
}

// ==========================================
// Base & Concrete OTP Context Models
// ==========================================
type BaseOTPInfo struct {
	Code           string
	PlatformChatID string
	TargetUserID   string
}

// ==========================================
// 1. Link OTP
// ==========================================

type LinkOTPInfo struct {
	BaseOTPInfo
	// just maybe
}

func WithLinkOTP(ctx context.Context, info LinkOTPInfo) context.Context {
	return context.WithValue(ctx, linkOTPCtxKey{}, info)
}

func GetLinkOTP(ctx context.Context) (LinkOTPInfo, bool) {
	info, ok := ctx.Value(linkOTPCtxKey{}).(LinkOTPInfo)
	return info, ok
}

func ParseLinkOTPMiddleware(parser *LinkCommandParser) bot.Middleware {
	return buildOTPMiddleware(parser, func(base BaseOTPInfo) LinkOTPInfo {
		return LinkOTPInfo{
			BaseOTPInfo: base,
		}
	}, WithLinkOTP)
}

// ==========================================
// 2. Deep Start Group Chat OTP
// ==========================================

type DeepStartGroupOTPInfo struct {
	BaseOTPInfo
	// just maybe

}

func WithDeepStartGroupOTP(ctx context.Context, info DeepStartGroupOTPInfo) context.Context {
	return context.WithValue(ctx, deepStartGroupOTPCtxKey{}, info)
}

func GetDeepStartGroupOTP(ctx context.Context) (DeepStartGroupOTPInfo, bool) {
	info, ok := ctx.Value(deepStartGroupOTPCtxKey{}).(DeepStartGroupOTPInfo)
	return info, ok
}

func ParseDeepStartGroupOTPMiddleware(parser *DeppStartGroupChatCompanyCommandParser) bot.Middleware {
	return buildOTPMiddleware(parser, func(base BaseOTPInfo) DeepStartGroupOTPInfo {
		return DeepStartGroupOTPInfo{
			BaseOTPInfo: base,
		}
	}, WithDeepStartGroupOTP)
}

// ==========================================
// 3. Deep Start Channel Chat OTP
// ==========================================

type DeepStartChannelOTPInfo struct {
	BaseOTPInfo
	// just maybe
}

func WithDeepStartChannelOTP(ctx context.Context, info DeepStartChannelOTPInfo) context.Context {
	return context.WithValue(ctx, deepStartChannelOTPCtxKey{}, info)
}

func GetDeepStartChannelOTP(ctx context.Context) (DeepStartChannelOTPInfo, bool) {
	info, ok := ctx.Value(deepStartChannelOTPCtxKey{}).(DeepStartChannelOTPInfo)
	return info, ok
}

func ParseDeepStartChannelOTPMiddleware(parser *DeppStartChannelChatCompanyCommandParser) bot.Middleware {
	return buildOTPMiddleware(parser, func(base BaseOTPInfo) DeepStartChannelOTPInfo {
		return DeepStartChannelOTPInfo{
			BaseOTPInfo: base,
		}
	}, WithDeepStartChannelOTP)
}

// ==========================================
// Generic bot.Middleware Runner (DRY Handler)
// ==========================================

func buildOTPMiddleware[T any](
	parser CommandParser,
	createInfo func(base BaseOTPInfo) T,
	withCtx func(context.Context, T) context.Context,
) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			msg, _ := telegram.ExtractMessage(update)
			text := msg.Content
			telChatID := msg.TargetID

			otpCode, err := parser.Parse(text)
			if err != nil {
				args := strings.Fields(strings.Trim(text, " "))
				errLogger.Error(exception.ErrBadRequest, "somthing in input text of feat chat with otp is not rights", logger.Any("args", args))

				_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: telChatID,
					Text:   "⚠️bad request⚠️",
				})
				if sendErr != nil {
					errLogger.Error(sendErr, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
					return
				}
				return
			}

			baseInfo := BaseOTPInfo{
				Code:           otpCode,
				PlatformChatID: fmt.Sprintf("%v", telChatID),
				TargetUserID:   msg.SenderID,
			}

			info := createInfo(baseInfo)
			ctx = withCtx(ctx, info)

			next(ctx, b, update)
		}
	}
}

// ====================================================================================
// Parsers
// ====================================================================================

type LinkCommandParser struct{}

func (p *LinkCommandParser) Parse(text string) (string, error) {
	text = strings.TrimSpace(text)
	args := strings.Fields(text)

	if len(args) != 2 || args[0] != "/link" {
		return "", exception.ErrBadRequest
	}

	return args[1], nil
}

// =========================================// =========================================// =========================================

type DeppStartGroupChatCompanyCommandParser struct {
	BaleUsername string
}

func (p *DeppStartGroupChatCompanyCommandParser) Parse(text string) (string, error) {
	text = strings.TrimSpace(text)

	args := strings.Fields(text)

	if len(args) != 2 {
		return "", exception.ErrBadRequest
	}

	cmd := args[0]

	expectedCmdWithUsername := fmt.Sprintf("/start@%s", strings.TrimPrefix(p.BaleUsername, "@"))

	if cmd != expectedCmdWithUsername {
		return "", exception.ErrBadRequest
	}

	return args[1], nil
}

// =========================================// =========================================// =========================================

type DeppStartChannelChatCompanyCommandParser struct{}

func (p *DeppStartChannelChatCompanyCommandParser) Parse(text string) (string, error) {
	text = strings.TrimSpace(text)
	args := strings.Fields(text)

	if len(args) != 2 || args[0] != "/start" {
		return "", exception.ErrBadRequest
	}

	return args[1], nil
}
