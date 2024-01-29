package models

type SessionStore struct {
	SessionId string
	UserId    int64
	Data      map[string]string
}

type SessionRepository interface {
	GetSession(session_id string) (*SessionStore, error)
	SaveSession(session *SessionStore) error
}
