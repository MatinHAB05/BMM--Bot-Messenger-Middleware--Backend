package repository_contract

type Repositories struct {
	AuthnTokenRepository
	ChatHistoryRepository
	ChatRepository
	CompanyRepository
	OTPRepository
	RateLimiterRepository
	RBACRepository
	UserRepository
}
