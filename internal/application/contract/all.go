package service_contract

import "time"

type Services struct {
	AttachmentService
	AuthService
	BroadcastService
	ChannelPendingService
	ChatLinkService
	ChatHistoryService
	ChatService
	CompanyService
	OTPService
	MediaGroupService
	RBACService
	SentBaleMsgService
	UserService
}

const timeLayout = time.RFC3339
