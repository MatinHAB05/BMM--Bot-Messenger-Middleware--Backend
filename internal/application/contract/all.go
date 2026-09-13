package service_contract

import "time"

type Services struct {
	AuthService
	BroadcastService
	ChatHistoryService
	ChatService
	CompanyService
	OTPService
	RBACService
	UserService
}

const timeLayout = time.RFC3339
