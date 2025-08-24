package middleware

import (
	"fmt"
	"net"
	"net/http"

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

func getSession(c *gin.Context, sm *SessionManager) *models.SessionStore {
	var session *models.SessionStore
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		qSessionId := c.Query("session_id")
		if qSessionId != "" {
			fmt.Println("DEBUG: Session loaded from query param", qSessionId)
			sessionID = qSessionId
		}
	} else {
		fmt.Println("DEBUG: SessionId loaded from cookie", sessionID)
	}

	if sessionID == "" {
		sessionID = betterguid.New()
		session = &models.SessionStore{SessionId: sessionID, UserId: sm.Node.Generate().Int64(), Data: map[string]string{}}
	} else {
		session, err = sm.SessionRepository.GetSession(sessionID)
		if err != nil {
			sessionID = betterguid.New()
			session = &models.SessionStore{SessionId: sessionID, UserId: sm.Node.Generate().Int64(), Data: map[string]string{}}
		}
	}
	return session
}

// func manage session to use as gin middleware
func (sm *SessionManager) ManageSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := getSession(c, sm)
		sessionID := session.SessionId
		c.Set("session", session)
		c.Set("session_mgr", sm)
		c.Next()
		if c.FullPath() == "" || c.Writer.Status() == http.StatusNotFound {
			fmt.Println("No session for this endpoint", c.FullPath())
			return
		}
		host := c.Request.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		fmt.Println("Setting cookie for", c.FullPath())
		c.SetCookie("session_id", sessionID, 3600*24*30, "/", host, false, false)
		sm.SessionRepository.SaveSession(session)
	}
}
