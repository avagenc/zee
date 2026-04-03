package chat

import (
	"context"
	"fmt"
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
		fmt.Printf("zep create thread (non-fatal, continuing statelessly): %v\n", err)
	}

	r.userThreadCache.Store(userID, threadID)
	return threadID, nil
}

func (r *repository) SaveMessages(ctx context.Context, threadID string, userMsg string, assistantMsg string) error {
	var messages []*zep.Message
	if userMsg != "" {
		messages = append(messages, &zep.Message{
			Role:    zep.RoleTypeUserRole,
			Content: userMsg,
		})
	}
	if assistantMsg != "" {
		messages = append(messages, &zep.Message{
			Role:    zep.RoleTypeAssistantRole,
			Content: assistantMsg,
		})
	}

	if len(messages) == 0 {
		return nil
	}

	_, err := r.zepClient.Thread.AddMessages(ctx, threadID, &zep.AddThreadMessagesRequest{
		Messages: messages,
	})
	return err
}
