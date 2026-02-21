package service

import (
	"errors"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/psds-microservice/session-manager-service/internal/errs"
	"github.com/psds-microservice/session-manager-service/internal/model"
	"gorm.io/gorm"
)

// SessionServicer — интерфейс для gRPC Deps (Dependency Inversion).
type SessionServicer interface {
	Create(clientID uuid.UUID, streamSessionID *uuid.UUID) (*model.ConsultationSession, error)
	GetByID(id uuid.UUID) (*model.ConsultationSession, error)
	GetParticipants(sessionID uuid.UUID) ([]model.SessionParticipant, error)
	IsParticipant(sessionID, userID uuid.UUID) (bool, error)
	JoinByPIN(pin string, operatorID uuid.UUID) (*model.ConsultationSession, error)
	JoinBySessionID(sessionID, operatorID uuid.UUID) (*model.ConsultationSession, error)
	Invite(sessionID, operatorID uuid.UUID) error
	Control(sessionID uuid.UUID, leadOperatorID *uuid.UUID, status string) error
	SetRecordingUrlByStreamSessionID(streamSessionID uuid.UUID, recordingURL string) error
}

const pinDigits = "0123456789"
const pinLength = 6

type SessionService struct {
	db *gorm.DB
}

func NewSessionService(db *gorm.DB) *SessionService {
	return &SessionService{db: db}
}

func generatePIN() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, pinLength)
	for i := range b {
		b[i] = pinDigits[r.Intn(len(pinDigits))]
	}
	return string(b)
}

func (s *SessionService) GetByID(id uuid.UUID) (*model.ConsultationSession, error) {
	var ses model.ConsultationSession
	if err := s.db.Where("id = ?", id).First(&ses).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrSessionNotFound
		}
		return nil, err
	}
	return &ses, nil
}

func (s *SessionService) GetByPIN(pin string) (*model.ConsultationSession, error) {
	var ses model.ConsultationSession
	if err := s.db.Where("pin = ? AND status != ?", pin, "finished").First(&ses).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrSessionNotFound
		}
		return nil, err
	}
	return &ses, nil
}

func (s *SessionService) JoinByPIN(pin string, operatorID uuid.UUID) (*model.ConsultationSession, error) {
	ses, err := s.GetByPIN(pin)
	if err != nil {
		return nil, err
	}
	return s.addParticipant(ses, operatorID, "operator")
}

func (s *SessionService) JoinBySessionID(sessionID uuid.UUID, operatorID uuid.UUID) (*model.ConsultationSession, error) {
	ses, err := s.GetByID(sessionID)
	if err != nil {
		return nil, err
	}
	if ses.Status == "finished" {
		return nil, errs.ErrSessionNotFound
	}
	return s.addParticipant(ses, operatorID, "operator")
}

func (s *SessionService) addParticipant(ses *model.ConsultationSession, userID uuid.UUID, role string) (*model.ConsultationSession, error) {
	var exists int64
	s.db.Model(&model.SessionParticipant{}).Where("session_id = ? AND user_id = ?", ses.ID, userID).Count(&exists)
	if exists > 0 {
		return ses, nil
	}
	p := model.SessionParticipant{SessionID: ses.ID, UserID: userID, Role: role, JoinedAt: time.Now()}
	if err := s.db.Create(&p).Error; err != nil {
		return nil, err
	}
	if ses.Status == "waiting" {
		ses.Status = "active"
		s.db.Model(ses).Update("status", "active")
	}
	return ses, nil
}

func (s *SessionService) Create(clientID uuid.UUID, streamSessionID *uuid.UUID) (*model.ConsultationSession, error) {
	pin := generatePIN()
	for i := 0; i < 10; i++ {
		var c int64
		s.db.Model(&model.ConsultationSession{}).Where("pin = ?", pin).Count(&c)
		if c == 0 {
			break
		}
		pin = generatePIN()
	}
	ses := model.ConsultationSession{
		ClientID:        clientID,
		PIN:             pin,
		Status:          "waiting",
		StreamSessionID: streamSessionID,
	}
	if err := s.db.Create(&ses).Error; err != nil {
		return nil, err
	}
	return &ses, nil
}

func (s *SessionService) Invite(sessionID uuid.UUID, operatorID uuid.UUID) error {
	ses, err := s.GetByID(sessionID)
	if err != nil {
		return err
	}
	if ses.Status == "finished" {
		return errs.ErrSessionNotFound
	}
	_, err = s.addParticipant(ses, operatorID, "operator")
	return err
}

func (s *SessionService) Control(sessionID uuid.UUID, leadOperatorID *uuid.UUID, status string) error {
	ses, err := s.GetByID(sessionID)
	if err != nil {
		return err
	}
	if ses.Status == "finished" {
		return errs.ErrSessionNotFound
	}
	upd := map[string]interface{}{}
	if leadOperatorID != nil {
		upd["lead_operator_id"] = leadOperatorID
	}
	if status != "" && (status == "active" || status == "finished") {
		upd["status"] = status
		if status == "finished" {
			now := time.Now()
			upd["finished_at"] = &now
		}
	}
	if len(upd) == 0 {
		return nil
	}
	return s.db.Model(ses).Updates(upd).Error
}

// SetRecordingUrlByStreamSessionID sets recording_url for the consultation session with the given stream_session_id.
func (s *SessionService) SetRecordingUrlByStreamSessionID(streamSessionID uuid.UUID, recordingURL string) error {
	res := s.db.Model(&model.ConsultationSession{}).Where("stream_session_id = ?", streamSessionID).Update("recording_url", recordingURL)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.ErrSessionNotFound
	}
	return nil
}

func (s *SessionService) GetParticipants(sessionID uuid.UUID) ([]model.SessionParticipant, error) {
	var list []model.SessionParticipant
	if err := s.db.Where("session_id = ?", sessionID).Order("joined_at").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// IsParticipant returns true if userID is the session client or a session operator.
func (s *SessionService) IsParticipant(sessionID, userID uuid.UUID) (bool, error) {
	ses, err := s.GetByID(sessionID)
	if err != nil {
		return false, err
	}
	if ses.ClientID == userID {
		return true, nil
	}
	var count int64
	if err := s.db.Model(&model.SessionParticipant{}).Where("session_id = ? AND user_id = ?", sessionID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
