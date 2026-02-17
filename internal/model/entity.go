package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConsultationSession — сессия консультации (бизнес-логика, не медиа).
type ConsultationSession struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClientID        uuid.UUID      `gorm:"type:uuid;not null" json:"client_id"`
	PIN             string         `gorm:"type:varchar(12);uniqueIndex;not null" json:"pin"`
	Status          string         `gorm:"type:varchar(20);not null;default:'waiting'" json:"status"` // waiting, active, finished
	LeadOperatorID  *uuid.UUID     `gorm:"type:uuid" json:"lead_operator_id,omitempty"`
	StreamSessionID *uuid.UUID     `gorm:"type:uuid" json:"stream_session_id,omitempty"` // связь с streaming-service
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	FinishedAt      *time.Time     `json:"finished_at,omitempty"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ConsultationSession) TableName() string { return "consultation_sessions" }

// SessionParticipant — участник сессии (оператор приглашён или присоединился).
type SessionParticipant struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Role      string    `gorm:"type:varchar(20);not null;default:'operator'" json:"role"` // operator, lead
	JoinedAt  time.Time `gorm:"not null" json:"joined_at"`
}

func (SessionParticipant) TableName() string { return "session_participants" }
