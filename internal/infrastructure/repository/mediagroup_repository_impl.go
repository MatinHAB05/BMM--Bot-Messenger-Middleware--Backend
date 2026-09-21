package repository

import (
	"context"
	"errors"
	"fmt"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	redisApt "messenger-backend/internal/infrastructure/redis"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// media_group_id:<mediaGroupID>:<chatID>  ->  <chatHistoryID>
type MediaGroupRepository struct {
	redisClient redisApt.Cache
}

func NewMediaGroupRepository(redisClient redisApt.Cache) repository_contract.MediaGroupRepository {
	return &MediaGroupRepository{redisClient: redisClient}
}

func mediaGroupKeyFunc(mediaGroupID string, chatID string) string {
	return fmt.Sprintf("media_group_id:%s:%s", mediaGroupID, chatID)
}

func formatChatHistoryID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func parseChatHistoryID(val string) (uint, error) {
	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid chat_history_id value %q for media group: %v", exception.ErrCacheOperation, val, err)
	}
	return uint(id), nil
}

// Set stores chatHistoryID for (mediaGroupID, chatID), overwriting any
// previous value, and (re)starts the TTL.
func (r *MediaGroupRepository) Set(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint, ttl time.Duration) error {
	key := mediaGroupKeyFunc(mediaGroupID, chatID)
	if err := r.redisClient.GetRDB().Set(ctx, key, formatChatHistoryID(chatHistoryID), ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}

// Get returns the chatHistoryID registered for (mediaGroupID, chatID), or
// exception.ErrMediaGroupMsgNotFound if there is none.
func (r *MediaGroupRepository) Get(ctx context.Context, mediaGroupID string, chatID string) (uint, error) {
	key := mediaGroupKeyFunc(mediaGroupID, chatID)
	val, err := r.redisClient.GetRDB().Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, exception.ErrMediaGroupMsgNotFound
		}
		return 0, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	return parseChatHistoryID(val)
}

func (r *MediaGroupRepository) Exists(ctx context.Context, mediaGroupID string, chatID string) (bool, error) {
	key := mediaGroupKeyFunc(mediaGroupID, chatID)
	count, err := r.redisClient.GetRDB().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return count > 0, nil
}

func (r *MediaGroupRepository) Delete(ctx context.Context, mediaGroupID string, chatID string) error {
	key := mediaGroupKeyFunc(mediaGroupID, chatID)
	res := r.redisClient.GetRDB().Del(ctx, key)
	if err := res.Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	if res.Val() == 0 {
		return exception.ErrMediaGroupMsgNotFound
	}

	return nil
}

func (r *MediaGroupRepository) GetAndDelete(ctx context.Context, mediaGroupID string, chatID string) (uint, error) {
	key := mediaGroupKeyFunc(mediaGroupID, chatID)
	val, err := r.redisClient.GetRDB().GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, exception.ErrMediaGroupMsgNotFound
		}
		return 0, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	return parseChatHistoryID(val)
}

// ForceGet is an atomic "get or claim": if (mediaGroupID, chatID) is
// already registered it returns the registered chatHistoryID; otherwise it
// registers the given chatHistoryID (with ttl) and returns it. SET NX makes
// the check-and-set a single Redis operation, so concurrent callers can't
// both "win".
func (r *MediaGroupRepository) ForceGet(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint, ttl time.Duration) (uint, error) {
	key := mediaGroupKeyFunc(mediaGroupID, chatID)

	claimed, err := r.redisClient.GetRDB().SetNX(ctx, key, formatChatHistoryID(chatHistoryID), ttl).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	if claimed {
		return chatHistoryID, nil
	}

	return r.Get(ctx, mediaGroupID, chatID)
}

func (r *MediaGroupRepository) TTL(ctx context.Context, mediaGroupID string, chatID string) (time.Duration, error) {
	key := mediaGroupKeyFunc(mediaGroupID, chatID)
	ttl, err := r.redisClient.GetRDB().TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	// go-redis surfaces Redis' "-2" reply (key does not exist) as a
	// Duration of -2.
	if ttl == -2 {
		return 0, exception.ErrMediaGroupMsgNotFound
	}

	return ttl, nil
}
