// service_contract/chat_link_service.go
package service_contract

import "context"

// ChatLinkService orchestrates the OTP-driven flows that connect a
// messenger chat or channel to a company. It exists to keep this
// multi-service coordination (OTP + Chat + ChannelPending) out of the
// presentation layer, so platform handlers (Telegram, Bale, ...) stay
// thin: pull data from middleware, call one method here, send a message.
type ChatLinkService interface {
	// LinkChatToCompany validates otpCode as a TypeLink OTP and assigns
	// the resulting companyID to the chat identified by platform +
	// platformChatID. Used by both the direct "/link" command and the
	// deep-start group flow.
	LinkChatToCompany(ctx context.Context, platform, platformChatID, otpCode string) (*ChatResponse, error)

	// PrepareChannelLink validates otpCode as a TypeLink OTP and stashes
	// the resulting companyID as "pending" against targetUserID, to be
	// claimed later once the bot is actually added to the channel
	// (see ConfirmChannelLink).
	PrepareChannelLink(ctx context.Context, otpCode, targetUserID string) error

	// ConfirmChannelLink consumes the pending link previously stored by
	// PrepareChannelLink for approverUserID and assigns the company to
	// the channel chat identified by platform + platformChatID.
	ConfirmChannelLink(ctx context.Context, platform, platformChatID, approverUserID string) (*ChatResponse, error)
}
