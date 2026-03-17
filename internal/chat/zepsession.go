package chat

import (
	"context"
	"iter"
	"log"
	"time"

	"github.com/getzep/zep-go/v3"
	zepclient "github.com/getzep/zep-go/v3/client"

	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const lastNMessages = 10

type zepSessionService struct {
	client *zepclient.Client
}

func NewZepSessionService(client *zepclient.Client) session.Service {
	return &zepSessionService{
		client: client,
	}
}

func (s *zepSessionService) Create(_ context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	return &session.CreateResponse{
		Session: &zepSessionImpl{
			id:     req.SessionID,
			userID: req.UserID,
			app:    req.AppName,
		},
	}, nil
}

func (s *zepSessionService) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	var events []*session.Event

	resp, err := s.client.Thread.Get(ctx, req.SessionID, &zep.ThreadGetRequest{
		Lastn: zep.Int(lastNMessages),
	})
	if err == nil && resp != nil {
		for _, msg := range resp.GetMessages() {
			if msg == nil {
				continue
			}
			role := "user"
			if msg.Role == zep.RoleTypeAssistantRole {
				role = "model"
			}

			uuidStr := ""
			if msg.UUID != nil {
				uuidStr = *msg.UUID
			}

			evt := session.NewEvent(uuidStr)
			evt.Author = role
			evt.LLMResponse = model.LLMResponse{
				Content: &genai.Content{
					Role:  role,
					Parts: []*genai.Part{{Text: msg.Content}},
				},
			}
			events = append(events, evt)
		}
	} else if err != nil {
		log.Printf("failed to fetch thread messages from zep: %v", err)
	}

	return &session.GetResponse{
		Session: &zepSessionImpl{
			id:     req.SessionID,
			userID: req.UserID,
			app:    req.AppName,
			events: events,
		},
	}, nil
}

func (s *zepSessionService) List(_ context.Context, _ *session.ListRequest) (*session.ListResponse, error) {
	return &session.ListResponse{}, nil
}

func (s *zepSessionService) Delete(_ context.Context, _ *session.DeleteRequest) error {
	return nil
}

func (s *zepSessionService) AppendEvent(_ context.Context, sess session.Session, event *session.Event) error {
	if impl, ok := sess.(*zepSessionImpl); ok {
		impl.events = append(impl.events, event)
	}

	sessionID := sess.ID()
	if sessionID == "" {
		return nil
	}

	content := event.LLMResponse.Content
	if content == nil {
		content = event.Content
	}
	if content == nil || len(content.Parts) == 0 {
		return nil
	}

	role := zep.RoleTypeUserRole
	if content.Role == "model" {
		role = zep.RoleTypeAssistantRole
	}

	var text string
	for _, p := range content.Parts {
		if p.Text != "" {
			text += p.Text
		}
	}
	if text == "" {
		return nil
	}

	_, err := s.client.Thread.AddMessages(context.Background(), sessionID, &zep.AddThreadMessagesRequest{
		Messages: []*zep.Message{{
			Role:    role,
			Content: text,
		}},
	})
	if err != nil {
		log.Printf("failed to append message to zep thread: %v", err)
	}

	return nil
}

type zepSessionImpl struct {
	id     string
	userID string
	app    string
	events []*session.Event
}

func (z *zepSessionImpl) ID() string                { return z.id }
func (z *zepSessionImpl) AppName() string           { return z.app }
func (z *zepSessionImpl) UserID() string            { return z.userID }
func (z *zepSessionImpl) LastUpdateTime() time.Time { return time.Now() }
func (z *zepSessionImpl) State() session.State      { return &zepStateImpl{} }
func (z *zepSessionImpl) Events() session.Events    { return &zepEventsImpl{events: z.events} }

type zepStateImpl struct{}

func (z *zepStateImpl) Get(_ string) (any, error)   { return nil, session.ErrStateKeyNotExist }
func (z *zepStateImpl) Set(_ string, _ any) error   { return nil }
func (z *zepStateImpl) All() iter.Seq2[string, any] { return func(func(string, any) bool) {} }

type zepEventsImpl struct {
	events []*session.Event
}

func (e *zepEventsImpl) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, evt := range e.events {
			if !yield(evt) {
				return
			}
		}
	}
}

func (e *zepEventsImpl) Len() int { return len(e.events) }

func (e *zepEventsImpl) At(i int) *session.Event {
	if i < 0 || i >= len(e.events) {
		return nil
	}
	return e.events[i]
}
