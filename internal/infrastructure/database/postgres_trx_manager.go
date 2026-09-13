package database

import (
	"context"

	"gorm.io/gorm"
)

type TrxManager interface {
	WithTransaction(ctx context.Context, fn func(trxCtx context.Context) error) error
}

type gormTrxManager struct {
	db Database
}

func NewTrxManager(db Database) TrxManager {
	return &gormTrxManager{db: db}
}

func (m *gormTrxManager) WithTransaction(ctx context.Context, fn func(trxCtx context.Context) error) error {
	return m.db.GetGormDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDatabase := m.db.WithTx(tx)
		trxCtx := InjectTrx(ctx, txDatabase)
		return fn(trxCtx)
	})
}
