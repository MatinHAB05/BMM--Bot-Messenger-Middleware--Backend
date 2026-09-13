package database

import "context"

type trxKey struct{}


func InjectTrx(ctx context.Context, db Database) context.Context {
	return context.WithValue(ctx, trxKey{}, db)
}


func ExtractTrxOrDB(ctx context.Context, defaultDB Database) Database {
	if db, ok := ctx.Value(trxKey{}).(Database); ok {
		return db
	}
	return defaultDB
}
