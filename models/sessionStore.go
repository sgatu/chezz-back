package models

type SessionStore struct {
	Data      map[string]string
	SessionId string
	UserId    int64
}

type SessionRepository interface {
	GetSession(session_id string) (*SessionStore, error)
	SaveSession(session *SessionStore) error
}
