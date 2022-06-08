package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func setSession(w http.ResponseWriter, session map[string]string) error {
	encoded, err := secureCookie.Encode("session", session)
	if err != nil {
		return err
	}
	c := &http.Cookie{
		Name:  "session",
		Value: encoded,
		Path:  "/",
	}
	http.SetCookie(w, c)
	return nil
}

func getSession(r *http.Request) (map[string]string, error) {
	value := make(map[string]string)
	c, _ := r.Cookie("session")
	if c == nil {
		return value, nil
	}
	err := secureCookie.Decode("session", c.Value, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func clearSession(w http.ResponseWriter) {
	c := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, c)
}

type AppSession struct {
	User    string
	Session string
}

// sessionCh is a channel to send/recv a session through AppSessionManager.
type sessionCh struct {
	ch      chan AppSession
	created time.Time
}

// AppSessionManager holds sessions for apps until they ask them.
// It helps app login process.
type AppSessionManager struct {
	sync.Mutex
	chs map[string]*sessionCh
}

// NewAppSessionManager creates a new AppSessionManager.
func NewAppSessionManager() *AppSessionManager {
	m := &AppSessionManager{
		chs: make(map[string]*sessionCh),
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			m.clearOldSessions()
		}
	}()
	return m
}

// DebugStatus prints the manager status every 5 seconds.
func (m *AppSessionManager) DebugStatus() {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			fmt.Printf("AppSessionManager is having %v sessions.\n", len(m.chs))
		}
	}()
}

// RecieveSession waits to recieve the session for given key.
// After 5 minutes has passed it will return timeout error.
func (m *AppSessionManager) RecieveSession(key string) (AppSession, error) {
	ch := m.recieveSessionCh(key)
	select {
	case sess := <-ch:
		m.deleteSession(key)
		return sess, nil
	case <-time.After(5 * time.Minute):
		return AppSession{}, fmt.Errorf("timeout")
	}
}

// recieveSessionCh returns channel for recieving the session for given key.
func (m *AppSessionManager) recieveSessionCh(key string) <-chan AppSession {
	m.Lock()
	defer m.Unlock()
	if m.chs[key] == nil {
		m.chs[key] = &sessionCh{
			ch:      make(chan AppSession),
			created: time.Now(),
		}
	}
	return m.chs[key].ch
}

// SendSession saves a session for given key.
func (m *AppSessionManager) SendSession(key string, sess AppSession) {
	m.Lock()
	defer m.Unlock()
	if m.chs[key] == nil {
		m.chs[key] = &sessionCh{
			ch:      make(chan AppSession),
			created: time.Now(),
		}
	}
	go func() {
		m.chs[key].ch <- sess
	}()
}

// deleteSession deletes a session for given key.
func (m *AppSessionManager) deleteSession(key string) {
	m.Lock()
	defer m.Unlock()
	close(m.chs[key].ch)
	delete(m.chs, key)
}

// clearOldSessions deletes sessions those are aged over 5 minutes.
func (m *AppSessionManager) clearOldSessions() {
	m.Lock()
	defer m.Unlock()
	now := time.Now()
	list := make([]string, 0)
	for key, as := range m.chs {
		if now.Sub(as.created) > 5*time.Minute {
			list = append(list, key)
		}
	}
	for _, key := range list {
		delete(m.chs, key)
	}
}
