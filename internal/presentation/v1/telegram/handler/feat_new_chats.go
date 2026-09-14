package telegramhandlers

// import (
// 	"context"
// 	"messenger-backend/internal/domain/entity"
// 	"messenger-backend/internal/domain/exception"
// 	"messenger-backend/internal/domain/otp"
// 	"messenger-backend/pkg/logger"
// 	"messenger-backend/pkg/messenger/telegram"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/go-telegram/bot"
// 	"github.com/go-telegram/bot/models"
// )

// func (h *TelegramIngestHandler) FeatChatWithOTP(ctx context.Context, b *bot.Bot, update *models.Update) {
// 	msg, _ := telegram.ExtractMessage(update)
// 	telChatID := msg.TargetID

// 	otpCode, _ := ctx.Value("otp_code").(string)
// 	otpChatType, _ := ctx.Value("otp_chat_type").(string)

// 	new_chat, err := h.chatService.FeatChatWithOTP(ctx, otpCode, telChatID, string(entity.PlatformTelegram), otpChatType)
// 	if err != nil || new_chat.CompanyID == nil {
// 		h.log.Error(err, "err from feat chat with otp", logger.Any("new_chat", new_chat))
// 		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
// 			ChatID: telChatID,
// 			Text:   "❌request failed❌",
// 		})
// 		if err != nil {
// 			h.log.Error(err, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
// 		}

// 		return
// 	}

// 	h.log.Debug("new_chat_resp", logger.Any("new_chat", new_chat))
// }

// func (h *TelegramIngestHandler) NewSendDoneHook() Hook {
// 	return Hook{
// 		After: func(ctx context.Context, b *bot.Bot, update *models.Update) (context.Context, error) {
// 			msg, _ := telegram.ExtractMessage(update)
// 			telChatID := msg.TargetID

// 			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
// 				ChatID: telChatID,
// 				Text:   "✅Done✅",
// 			})
// 			if err != nil {
// 				h.log.Error(err, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
// 				return ctx, err
// 			}
// 			return ctx, nil

// 		},
// 	}
// }

// func (h *TelegramIngestHandler) NewSetPendingChannelRegistraionHook() Hook {
// 	return Hook{
// 		After: func(ctx context.Context, b *bot.Bot, update *models.Update) (context.Context, error) {
// 			msg, _ := telegram.ExtractMessage(update)

// 			otpCode, _ := ctx.Value("otp_code").(string)
// 			otpChatType := ctx.Value("otp_chat_type").(string)
// 			h.otpService.InvalidateOTPAndSetVerified(ctx, "==dose not matter==", otp.Type(otpChatType), otpCode)

// 			rawPayload, err := h.otpService.VerifyOTP(ctx, "==does not matter==", otp.Type(otpChatType), otpCode)
// 			if err != nil {
// 				return nil, exception.Wrap(exception.ErrInternal, err)
// 			}

// 			payload, ok := rawPayload.(*otp.LinkChatCompanyPayload)
// 			if !ok {
// 				h.log.Warn("fail to type assert raw-payload for regiser-user-otp", logger.Any("raw-payload", rawPayload))
// 				return nil, exception.Wrap(exception.ErrInternal, err)
// 			}
// 			uint64_com_id, err := strconv.ParseUint(payload.CompanyID, 10, 64)
// 			if err != nil {
// 				return nil, exception.Wrap(exception.ErrInternal, err)
// 			}

// 			err = h.channelpendingService.SetPendingChannel(ctx, msg.SenderID, uint(uint64_com_id), time.Duration(time.Minute*5))
// 			if err != nil {
// 				return nil, exception.Wrap(exception.ErrInternal, err)

// 			}
// 			return ctx, nil
// 		},
// 	}
// }

// // func (h *TelegramIngestHandler) NewOTPParseHook(parser CommandParser) Hook {
// // 	return Hook{
// // 		Before: func(ctx context.Context, b *bot.Bot, update *models.Update) (context.Context, error) {
// // 			msg, _ := telegram.ExtractMessage(update)
// // 			text := msg.Content
// // 			telChatID := msg.TargetID

// // 			otpCode, err := parser.Parse(text)
// // 			if err != nil {
// // 				args := strings.Split(strings.Trim(text, " "), " ")
// // 				h.log.Error(exception.ErrBadRequest, "somthing in input text of feat chat with otp is not rights", logger.Any("args", args))

// // 				_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
// // 					ChatID: telChatID,
// // 					Text:   "⚠️bad request⚠️",
// // 				})
// // 				if sendErr != nil {
// // 					h.log.Error(sendErr, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
// // 				}
// // 				return ctx, err
// // 			}
// // 			ctx = context.WithValue(ctx, "otp_code", otpCode)
// // 			ctx = context.WithValue(ctx, "otp_chat_type", parser.ChatType())
// // 			return ctx, nil
// // 		},
// // 	}
// // }

// func (h *TelegramIngestHandler) NewOTPSendInlineKeyboardHook() Hook {
// 	return Hook{
// 		After: func(ctx context.Context, b *bot.Bot, update *models.Update) (context.Context, error) {
// 			msg, _ := telegram.ExtractMessage(update)
// 			telChatID := msg.TargetID

// 			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
// 				ChatID: telChatID,
// 				Text:   "Welcome!\nWe're ready to connect your Telegram channel to your company dashboard.\n\nClick the button below to select your channel and grant the bot administrative access.",
// 				ReplyMarkup: models.InlineKeyboardMarkup{
// 					InlineKeyboard: [][]models.InlineKeyboardButton{
// 						{
// 							{Text: "📢 Select & Add Channel", CallbackData: "==dose not matter=="},
// 						},
// 					},
// 				},
// 			})

// 			if err != nil {
// 				h.log.Error(err, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
// 				return ctx, err
// 			}
// 			return ctx, nil
// 		},
// 	}
// }

// func (h *TelegramIngestHandler) FeatJustChannelChatJoinChannel(ctx context.Context, b *bot.Bot, update *models.Update) {
// 	telChatID, inviterID, ok := telegram.ExtractBotJoinedChannelInfo(update)
// 	if !ok {
// 		h.log.Error(exception.ErrFalsyHandlerRouting, "handler routing has not worked well - FeatJustChannelChatJoinChannel", logger.Any("chat-id", telChatID))
// 		return
// 	}
// 	companyID, err := h.channelpendingService.GetPendingChannel(ctx, strconv.FormatInt(inviterID, 10))
// 	if err != nil {
// 		h.log.Error(err, exception.ErrInternal.Error(), logger.Any("chat-id", telChatID))

// 		_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
// 			ChatID: telChatID,
// 			Text:   "Session Expired\nThis connection link has expired.\n\nPlease go back to your web dashboard and click 'Connect Channel' to get a new link.",
// 		})
// 		if sendErr != nil {
// 			h.log.Error(sendErr, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
// 			return
// 		}
// 		return
// 	}

// 	chat, err := h.chatService.GetByPlatformID(ctx, string(entity.PlatformTelegram), strconv.FormatInt(inviterID, 10))

// 	h.log.Debug("chat begin", logger.Any("chat", chat))

// 	if err != nil {
// 		h.log.Error(err, exception.ErrInternal.Error(), logger.Any("chat-id", telChatID))
// 		return
// 	} else if chat.CompanyID != nil {
// 		h.log.Error(exception.ErrChatCompanyAlreadyRegistered, "chat was already regestred by a company", logger.Uint("chat-id", chat.ID), logger.Any("current_company_id", chat.CompanyID))
// 		return
// 	}

// 	h.log.Debug("before update : ", logger.Any("new_chat", chat))

// 	chatResp, err := h.chatService.UpdateCompanyID(ctx, companyID, chat.ID)
// 	if err != nil {
// 		h.log.Error(err, exception.ErrInternal.Error(), logger.Any("chat-id", telChatID))

// 		return
// 	}
// 	h.log.Debug("after update : ", logger.Any("new_chat", chatResp))
// }
