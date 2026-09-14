package apihandler

type APIHandlers struct {
	*AuthHandler
	*BroadcastHandler
	*ChatHandler
	*ChatHistoryHandler
	*CompanyHandler
	*UserHandler
}
