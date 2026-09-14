package handler

import (
	apihandler "messenger-backend/internal/presentation/v1/api/handler"
	balehandlers "messenger-backend/internal/presentation/v1/bale/handler"
	telegramhandlers "messenger-backend/internal/presentation/v1/telegram/handler"
)

type Handlers struct {
	*apihandler.APIHandlers
	*telegramhandlers.TelegramHandlers
	*balehandlers.BaleHandlers
}
