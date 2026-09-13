package handler

type Handlers struct {
	*AuthHandler
	*BroadcastHandler
	*ChatHandler
	*ChatHistoryHandler
	*CompanyHandler
	*TelegramIngestHandler
	*UserHandler
}
