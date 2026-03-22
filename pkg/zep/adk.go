package zep

import (
	"context"
	"iter"
	"log"
	"strings"
	"time"

	"github.com/getzep/zep-go/v3"
	"github.com/getzep/zep-go/v3/client"

	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type adkSessionService struct {
	client              *client.Client
	agentName           string
	contextWindowLength int
}

func NewADKSessionService(client *client.Client, agentName string, contextWindowLength int) *adkSessionService {
	return &adkSessionService{
		client:              client,
		agentName:           agentName,
		contextWindowLength: contextWindowLength,
	}
}

func (s *adkSessionService) Create(_ context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	return &session.CreateResponse{
		Session: &zepSession{
			id:     req.SessionID,
			userID: req.UserID,
			app:    req.AppName,
		},
	}, nil
}

func (s *adkSessionService) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	sess := &zepSession{
		id:     req.SessionID,
		userID: req.UserID,
		app:    req.AppName,
	}

	resp, err := s.client.Thread.Get(ctx, req.SessionID, &zep.ThreadGetRequest{
		Lastn: zep.Int(s.contextWindowLength),
	})
	if err != nil {
		log.Printf("failed to fetch thread messages from zep: %v", err)
		return &session.GetResponse{Session: sess}, nil
	}

	contextResp, ctxErr := s.client.Thread.GetUserContext(ctx, req.SessionID, &zep.ThreadGetUserContextRequest{})
	if ctxErr != nil {
		log.Printf("failed to fetch user context from zep for session %s: %v", req.SessionID, ctxErr)
	} else if contextResp != nil && contextResp.GetContext() != nil {
		ctxStr := *contextResp.GetContext()
		if ctxStr != "" {
			evt := session.NewEvent("context-injection")
			evt.Author = "user"
			evt.LLMResponse = model.LLMResponse{
				Content: genai.NewContentFromText(ctxStr, genai.Role("user")),
			}
			sess.events = append(sess.events, evt)
		}
	}

	for _, msg := range resp.GetMessages() {
		if msg == nil {
			continue
		}

		role := s.zepRoleToADK(msg.Role)
		evt := session.NewEvent(derefOrEmpty(msg.UUID))
		evt.Author = role

		contentRole := "model"
		if role == "user" {
			contentRole = "user"
		}

		evt.LLMResponse = model.LLMResponse{
			Content: genai.NewContentFromText(msg.Content, genai.Role(contentRole)),
		}
		sess.events = append(sess.events, evt)
	}

	return &session.GetResponse{Session: sess}, nil
}

func (s *adkSessionService) List(_ context.Context, _ *session.ListRequest) (*session.ListResponse, error) {
	return &session.ListResponse{}, nil
}

func (s *adkSessionService) Delete(_ context.Context, _ *session.DeleteRequest) error {
	return nil
}

func (s *adkSessionService) AppendEvent(ctx context.Context, sess session.Session, event *session.Event) error {
	if impl, ok := sess.(*zepSession); ok {
		impl.events = append(impl.events, event)
	}

	if sess.ID() == "" {
		return nil
	}

	content := event.LLMResponse.Content
	if content == nil || len(content.Parts) == 0 {
		return nil
	}

	if containsFunctionParts(content.Parts) {
		return nil
	}

	text := extractText(content.Parts)
	if text == "" {
		return nil
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := s.client.Thread.AddMessages(bgCtx, sess.ID(), &zep.AddThreadMessagesRequest{
			Messages: []*zep.Message{{
				Role:    s.adkRoleToZep(string(content.Role)),
				Content: text,
			}},
		})
		if err != nil {
			log.Printf("failed to append message to zep thread async: %v", err)
		}
	}()

	return nil
}

func containsFunctionParts(parts []*genai.Part) bool {
	for _, p := range parts {
		if p.FunctionCall != nil || p.FunctionResponse != nil {
			return true
		}
	}
	return false
}

func (s *adkSessionService) zepRoleToADK(role zep.RoleType) string {
	if role == zep.RoleTypeAssistantRole {
		return s.agentName
	}
	return "user"
}

func (s *adkSessionService) adkRoleToZep(role string) zep.RoleType {
	if role == "model" || role == s.agentName {
		return zep.RoleTypeAssistantRole
	}
	return zep.RoleTypeUserRole
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func extractText(parts []*genai.Part) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Text)
	}
	return b.String()
}

type zepSession struct {
	id     string
	userID string
	app    string
	events []*session.Event
}

func (z *zepSession) ID() string                { return z.id }
func (z *zepSession) AppName() string           { return z.app }
func (z *zepSession) UserID() string            { return z.userID }
func (z *zepSession) LastUpdateTime() time.Time { return time.Now() }
func (z *zepSession) State() session.State      { return zepState{} }
func (z *zepSession) Events() session.Events    { return zepEvents(z.events) }

type zepState struct{}

func (zepState) Get(_ string) (any, error)   { return nil, session.ErrStateKeyNotExist }
func (zepState) Set(_ string, _ any) error   { return nil }
func (zepState) All() iter.Seq2[string, any] { return func(func(string, any) bool) {} }

type zepEvents []*session.Event

func (e zepEvents) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, evt := range e {
			if !yield(evt) {
				return
			}
		}
	}
}

func (e zepEvents) Len() int { return len(e) }

func (e zepEvents) At(i int) *session.Event {
	if i < 0 || i >= len(e) {
		return nil
	}
	return e[i]
}
