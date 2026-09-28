package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	redisApt "messenger-backend/internal/infrastructure/redis"

	"github.com/redis/go-redis/v9"
)

// otpRepository implements repository_contract.OTPRepository on top of
// Redis. An OTP is just a short-lived key/value pair -- the key is
// computed by the caller's otp.Strategy, the value is whatever JSON
// bytes OTPService serialized the Payload into -- so there's no
// PostgreSQL involvement here, same as before.
//
// This replaces the previous implementation, which stored payloads via
// HSet with a *value* argument where an EXPIRE duration was clearly
// intended (so keys never actually expired), and compared an HGetAll
// map[string]string result as raw bytes (which doesn't type-check as
// written). A plain SET with an explicit TTL sidesteps both problems and
// is also a better fit now that a payload is arbitrary JSON rather than
// a single hash field.
type otpRepository struct {
	redisClient redisApt.Cache
}

// NewOTPRepository builds the Redis-backed OTPRepository.
func NewOTPRepository(redisClient redisApt.Cache) repository_contract.OTPRepository {
	return &otpRepository{redisClient: redisClient}
}

func (r *otpRepository) Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	if err := r.rdb(ctx).Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}

func verifedKey(key string) string {
	return key + ":verified"
}

func (r *otpRepository) SetVerfied(ctx context.Context, key string, ttl time.Duration) error {
	if err := r.rdb(ctx).Set(ctx, verifedKey(key), true, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}

func (r *otpRepository) IsVerified(ctx context.Context, key string) (*bool, error) {
	var is bool
	val, err := r.rdb(ctx).Get(ctx, verifedKey(key)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			is := false
			return &is, nil
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	is = val == "true" || val == "1"
	return &is, nil
}

func (r *otpRepository) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.rdb(ctx).Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, exception.ErrOTPNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return val, nil
}

func (r *otpRepository) Delete(ctx context.Context, key string) error {
	res := r.rdb(ctx).Del(ctx, key)
	if err := res.Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	if res.Val() == 0 {
		return exception.ErrOTPNotFound
	}

	return nil
}

func (r *otpRepository) DeleteVerified(ctx context.Context, key string) error {
	res := r.rdb(ctx).Del(ctx, verifedKey(key))
	if err := res.Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	if res.Val() == 0 {
		return exception.ErrOTPNotFound
	}

	return nil
}

func (r *otpRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.rdb(ctx).Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return count > 0, nil
}

func (r *otpRepository) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.rdb(ctx).TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	// go-redis returns -2 when the key doesn't exist and -1 when it
	// exists but carries no expiry; every key this repository writes
	// always has a TTL, so treat any negative result as "not found."
	if ttl < 0 {
		return 0, exception.ErrOTPNotFound
	}

	return ttl, nil
}

func (r *otpRepository) rdb(ctx context.Context) redis.Cmdable {
	return redisApt.ExtractTrxOrCache(ctx, r.redisClient).GetRDB()
}

// VerifiedKey exposes the key under which the "verified" flag of an OTP key
// is stored (for WithWatch). The OTP key itself comes from the otp.Strategy.
func (r *otpRepository) VerifiedKey(key string) string {
	return verifedKey(key)
}

// Remove deletes the OTP key without inspecting the reply (safe in a trx write phase).
func (r *otpRepository) Remove(ctx context.Context, key string) error {
	if err := r.rdb(ctx).Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}

// RemoveVerified deletes the "verified" flag without inspecting the reply.
func (r *otpRepository) RemoveVerified(ctx context.Context, key string) error {
	if err := r.rdb(ctx).Del(ctx, verifedKey(key)).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}
