package chat

import (
	"context"
	"log"
	"sync"

	"github.com/getzep/zep-go/v3"
	"github.com/getzep/zep-go/v3/client"
	"github.com/google/uuid"
)

type repository struct {
	zepClient       *client.Client
	userThreadCache sync.Map
}

func NewRepository(zepClient *client.Client) *repository {
	return &repository{
		zepClient: zepClient,
	}
}

func (r *repository) UpsertUser(ctx context.Context, userID string) error {
	_, err := r.zepClient.User.Add(ctx, &zep.CreateUserRequest{
		UserID: userID,
	})
	return err
}

func (r *repository) GetOrCreateThreadID(ctx context.Context, userID string) (string, error) {
	if cachedID, ok := r.userThreadCache.Load(userID); ok {
		return cachedID.(string), nil
	}

	threads, err := r.zepClient.User.GetThreads(ctx, userID)
	if err == nil && len(threads) > 0 {
		threadID := *threads[0].ThreadID
		r.userThreadCache.Store(userID, threadID)
		return threadID, nil
	}

	threadID := uuid.New().String()
	_, err = r.zepClient.Thread.Create(ctx, &zep.CreateThreadRequest{
		ThreadID: threadID,
		UserID:   userID,
	})
	if err != nil {
		log.Printf("zep create thread (non-fatal, continuing statelessly): %v", err)
	}

	r.userThreadCache.Store(userID, threadID)
	return threadID, nil
}
