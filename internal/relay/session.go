package relay

import (
	"log/slog"
	"sync"

	"kiro-relay/internal/kiro"
)

// Session tracks per-user kiro session state.
type Session struct {
	HasHistory bool // true after first message, enables --resume
}

// SessionManager provides concurrency-safe session tracking shared across platforms.
type SessionManager struct {
	kiro     *kiro.Client
	sessions map[string]Session // key: "platform:userID"
	mu       sync.Mutex
}

func NewSessionManager(kiro *kiro.Client) *SessionManager {
	return &SessionManager{
		kiro:     kiro,
		sessions: make(map[string]Session),
	}
}

func (sm *SessionManager) Start(key string) {
	sm.mu.Lock()
	sm.sessions[key] = Session{}
	sm.mu.Unlock()
}

func (sm *SessionManager) End(key string) {
	sm.mu.Lock()
	delete(sm.sessions, key)
	sm.mu.Unlock()
}

func (sm *SessionManager) Get(key string) (Session, bool) {
	sm.mu.Lock()
	s, ok := sm.sessions[key]
	sm.mu.Unlock()
	return s, ok
}

// Send sends a prompt to kiro-cli and returns the response.
// It automatically manages the --resume flag based on session state.
func (sm *SessionManager) Send(key, prompt string) (string, error) {
	sm.mu.Lock()
	s, ok := sm.sessions[key]
	sm.mu.Unlock()

	if !ok {
		return "", nil
	}

	slog.Info("request", "session", key, "prompt", prompt)

	resp, err := sm.kiro.Run(prompt, s.HasHistory)
	if err != nil {
		slog.Error("response error", "session", key, "error", err)
		return "", err
	}

	slog.Info("response", "session", key, "response", resp)

	sm.mu.Lock()
	if sess, exists := sm.sessions[key]; exists {
		sess.HasHistory = true
		sm.sessions[key] = sess
	}
	sm.mu.Unlock()

	return resp, nil
}

func (sm *SessionManager) SetWorkDir(dir string) {
	sm.kiro.SetWorkDir(dir)
}

func (sm *SessionManager) GetWorkDir() string {
	return sm.kiro.GetWorkDir()
}
