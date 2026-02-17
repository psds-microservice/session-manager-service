package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/psds-microservice/session-manager-service/internal/errs"
	"github.com/psds-microservice/session-manager-service/internal/service"
)

type SessionHandler struct {
	svc *service.SessionService
}

func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

// GetSession GET /session/:id
func (h *SessionHandler) GetSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ses, err := h.svc.GetByID(id)
	if err != nil {
		if errors.Is(err, errs.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ses)
}

// JoinSession POST /session/join (body: session_id or pin, user_id)
func (h *SessionHandler) JoinSession(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id"`
		PIN       string `json:"pin"`
		UserID    string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	var ses interface{}
	if req.PIN != "" {
		ses, err = h.svc.JoinByPIN(req.PIN, userID)
	} else if req.SessionID != "" {
		sid, _ := uuid.Parse(req.SessionID)
		ses, err = h.svc.JoinBySessionID(sid, userID)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id or pin required"})
		return
	}
	if err != nil {
		if errors.Is(err, errs.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ses)
}

// Invite POST /session/:id/invite (body: operator_id)
func (h *SessionHandler) Invite(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		OperatorID string `json:"operator_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OperatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "operator_id required"})
		return
	}
	opID, _ := uuid.Parse(req.OperatorID)
	if err := h.svc.Invite(sessionID, opID); err != nil {
		if errors.Is(err, errs.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Control POST /session/:id/control (body: lead_operator_id?, status?)
func (h *SessionHandler) Control(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		LeadOperatorID string `json:"lead_operator_id"`
		Status         string `json:"status"`
	}
	_ = c.ShouldBindJSON(&req)
	var leadID *uuid.UUID
	if req.LeadOperatorID != "" {
		id, _ := uuid.Parse(req.LeadOperatorID)
		leadID = &id
	}
	if err := h.svc.Control(sessionID, leadID, req.Status); err != nil {
		if errors.Is(err, errs.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetParticipants GET /session/:id/participants
func (h *SessionHandler) GetParticipants(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	list, err := h.svc.GetParticipants(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}
