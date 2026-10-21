package bot

import (
	"context"
	"encoding/json"
	"sync"
)

type UserSession struct {
	UserId    string `json:"user_id"`
	SessionId string `json:"session_id"`
	PublicKey string `json:"public_key"`
	Platform  string `json:"platform"`
}

// SessionStore caches the encryption sessions for a recipient user.
// Get should return nil, nil when the user is not in the store.
type SessionStore interface {
	Get(userID string) ([]*Session, error)
	Put(userID string, sessions []*Session) error
	Delete(userID string) error
}

// MapSessionStore is an in-memory SessionStore implementation.
type MapSessionStore struct {
	mu       sync.RWMutex
	sessions map[string][]*Session
}

func NewMapSessionStore() *MapSessionStore {
	return &MapSessionStore{sessions: make(map[string][]*Session)}
}

func (s *MapSessionStore) Get(userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSessions(s.sessions[userID]), nil
}

func (s *MapSessionStore) Put(userID string, sessions []*Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions == nil {
		s.sessions = make(map[string][]*Session)
	}
	s.sessions[userID] = cloneSessions(sessions)
	return nil
}

func (s *MapSessionStore) Delete(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, userID)
	return nil
}

func cloneSessions(sessions []*Session) []*Session {
	if sessions == nil {
		return nil
	}
	cloned := make([]*Session, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		copy := *session
		cloned = append(cloned, &copy)
	}
	return cloned
}

func FetchUserSessions(ctx context.Context, users []string, su *SafeUser) ([]*UserSession, error) {
	data, err := json.Marshal(users)
	if err != nil {
		return nil, err
	}
	method, path := "POST", "/sessions/fetch"
	token, err := SignAuthenticationToken(method, path, string(data), su)
	if err != nil {
		return nil, err
	}
	body, err := Request(ctx, "POST", path, data, token)
	if err != nil {
		return nil, ServerError(ctx, err)
	}
	var resp struct {
		Data  []*UserSession `json:"data"`
		Error Error          `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, BadDataError(ctx)
	}
	if resp.Error.Code > 0 {
		return nil, resp.Error
	}
	return resp.Data, nil
}
