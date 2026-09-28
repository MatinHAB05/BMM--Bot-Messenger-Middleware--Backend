package redis

import "context"

type trxKey struct{}

func InjectTrx(ctx context.Context, c Cache) context.Context {
	return context.WithValue(ctx, trxKey{}, c)
}

// ExtractTrxOrCache returns the transactional Cache if ctx carries one,
// otherwise defaultCache.
func ExtractTrxOrCache(ctx context.Context, defaultCache Cache) Cache {
	if c, ok := ctx.Value(trxKey{}).(Cache); ok {
		return c
	}
	return defaultCache
}
