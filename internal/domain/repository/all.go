package repository_contract

type Repositories struct {
	AuthnTokenRepository
	ChannelPendingRepository
	ChatHistoryRepository
	ChatRepository
	CompanyRepository
	OTPRepository
	RateLimiterRepository
	RBACRepository
	SentBaleMsgRepository
	UserRepository
}
