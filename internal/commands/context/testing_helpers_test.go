package ctxcmd

import (
	"context"
	"iter"
	"sync"

	"github.com/mrlm-net/cure/pkg/agent"
)

// mockAgent is a test double for agent.Agent.
// Configure events and/or err before calling Run.
type mockAgent struct {
	events []agent.Event
	err    error
}

func (m *mockAgent) Run(_ context.Context, _ *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		for _, ev := range m.events {
			if !yield(ev, nil) {
				return
			}
		}
		if m.err != nil {
			yield(agent.Event{Kind: agent.EventKindError, Err: m.err.Error()}, m.err)
		}
	}
}

func (m *mockAgent) CountTokens(_ context.Context, _ *agent.Session) (int, error) {
	return 0, agent.ErrCountNotSupported
}

func (m *mockAgent) Provider() string { return "mock" }

// mockStore is a minimal in-memory implementation of agent.SessionStore.
type mockStore struct {
	mu       sync.Mutex
	sessions map[string]*agent.Session
	saveErr  error
	loadErr  error
	forkErr  error
}

func newMockStore() *mockStore {
	return &mockStore{sessions: make(map[string]*agent.Session)}
}

func (s *mockStore) Save(_ context.Context, sess *agent.Session) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Deep-copy history to avoid aliasing.
	history := make([]agent.Message, len(sess.History))
	copy(history, sess.History)
	tags := make([]string, len(sess.Tags))
	copy(tags, sess.Tags)
	cp := *sess
	cp.History = history
	cp.Tags = tags
	s.sessions[sess.ID] = &cp
	return nil
}

func (s *mockStore) Load(_ context.Context, id string) (*agent.Session, error) {
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, agent.ErrSessionNotFound
	}
	cp := *sess
	return &cp, nil
}

func (s *mockStore) List(_ context.Context) ([]*agent.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*agent.Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		cp := *sess
		out = append(out, &cp)
	}
	return out, nil
}

func (s *mockStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[id]; !ok {
		return agent.ErrSessionNotFound
	}
	delete(s.sessions, id)
	return nil
}

func (s *mockStore) Fork(_ context.Context, id string) (*agent.Session, error) {
	if s.forkErr != nil {
		return nil, s.forkErr
	}
	s.mu.Lock()
	src, ok := s.sessions[id]
	if !ok {
		s.mu.Unlock()
		return nil, agent.ErrSessionNotFound
	}
	forked := src.Fork()
	s.mu.Unlock()
	_ = s.Save(context.Background(), forked)
	return forked, nil
}
