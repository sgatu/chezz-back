package middleware

import (
	"net"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/kjk/betterguid"
	"github.com/sgatu/chezz-back/models"
)

type SessionManager struct {
	SessionRepository models.SessionRepository
	Node              *snowflake.Node
}

func NewSessionManager(sessionRepository models.SessionRepository) *SessionManager {
	return &SessionManager{
		SessionRepository: sessionRepository,
	}
}

// func to set data in session
func (sm *SessionManager) SetSessionData(session *models.SessionStore, key string, value string) {
	session.Data[key] = value
	sm.SessionRepository.SaveSession(session)
}

// func to remove data in session
func (sm *SessionManager) RemoveSessionData(session *models.SessionStore, key string) {
	delete(session.Data, key)
	sm.SessionRepository.SaveSession(session)
}

// func manage session to use as gin middleware
func (sm *SessionManager) ManageSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		var session *models.SessionStore
		if err != nil {
			sessionID = betterguid.New()
			session = &models.SessionStore{SessionId: sessionID, UserId: sm.Node.Generate().Int64(), Data: map[string]string{}}
		} else {
			session, err = sm.SessionRepository.GetSession(sessionID)
			if err != nil {
				sessionID = betterguid.New()
				session = &models.SessionStore{SessionId: sessionID, UserId: sm.Node.Generate().Int64(), Data: map[string]string{}}
			}
		}
		host, _, _ := net.SplitHostPort(c.Request.Host)
		c.SetCookie("session_id", sessionID, 3600*24*30, "/", host, false, true)
		c.Set("session", session)
		c.Set("session_mgr", sm)
		sm.SessionRepository.SaveSession(session)
	}
}
