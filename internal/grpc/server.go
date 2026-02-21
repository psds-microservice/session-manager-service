package grpc

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/psds-microservice/session-manager-service/internal/errs"
	"github.com/psds-microservice/session-manager-service/internal/kafka"
	"github.com/psds-microservice/session-manager-service/internal/model"
	"github.com/psds-microservice/session-manager-service/internal/searchindex"
	"github.com/psds-microservice/session-manager-service/internal/service"
	"github.com/psds-microservice/session-manager-service/pkg/gen/session_manager_service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Deps — зависимости gRPC-сервера (D: зависимость от абстракций).
type Deps struct {
	Session  service.SessionServicer
	Indexer  searchindex.SessionIndexer // опционально: индексация сессий в search-service
	Producer kafka.SessionEventProducer // опционально: события сессий в Kafka
}

// Server implements session_manager_service.SessionManagerServiceServer
type Server struct {
	session_manager_service.UnimplementedSessionManagerServiceServer
	Deps
}

// NewServer создаёт gRPC-сервер с внедрёнными сервисами
func NewServer(deps Deps) *Server {
	return &Server{Deps: deps}
}

// getMetadata returns the first value for key from incoming gRPC metadata (case-insensitive key).
func getMetadata(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	keyLower := strings.ToLower(key)
	if v := md.Get(keyLower); len(v) > 0 {
		return strings.TrimSpace(v[0])
	}
	return ""
}

func (s *Server) mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errs.ErrSessionNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, errs.ErrInvalidPIN) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	log.Printf("grpc: unhandled error: %v", err)
	return status.Error(codes.Internal, err.Error())
}

func toProtoSession(ses *model.ConsultationSession) *session_manager_service.SessionResponse {
	if ses == nil {
		return nil
	}
	return &session_manager_service.SessionResponse{
		Id:           ses.ID.String(),
		Status:       ses.Status,
		Pin:          ses.PIN,
		RecordingUrl: ses.RecordingURL,
	}
}

func (s *Server) CreateSession(ctx context.Context, req *session_manager_service.CreateSessionRequest) (*session_manager_service.SessionResponse, error) {
	clientID, err := uuid.Parse(req.GetClientId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid client_id")
	}
	var streamSessionID *uuid.UUID
	if req.GetStreamSessionId() != "" {
		parsed, err := uuid.Parse(req.GetStreamSessionId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid stream_session_id")
		}
		streamSessionID = &parsed
	}
	ses, err := s.Session.Create(clientID, streamSessionID)
	if err != nil {
		return nil, s.mapError(err)
	}
	// Отправляем событие в Kafka для индексации в search-service
	if s.Producer != nil {
		go s.Producer.ProduceSessionEvent(ctx, "session.created", ses.ID, map[string]interface{}{
			"client_id": ses.ClientID.String(),
			"pin":       ses.PIN,
			"status":    ses.Status,
		})
	}
	return toProtoSession(ses), nil
}

func (s *Server) GetSession(ctx context.Context, req *session_manager_service.GetSessionRequest) (*session_manager_service.SessionResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	ses, err := s.Session.GetByID(id)
	if err != nil {
		return nil, s.mapError(err)
	}
	// Индексация теперь через Kafka consumer в search-service worker
	return toProtoSession(ses), nil
}

func (s *Server) GetParticipants(ctx context.Context, req *session_manager_service.GetParticipantsRequest) (*session_manager_service.ParticipantsResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	// Проверяем существование сессии перед получением участников
	_, err = s.Session.GetByID(id)
	if err != nil {
		return nil, s.mapError(err)
	}
	participants, err := s.Session.GetParticipants(id)
	if err != nil {
		return nil, s.mapError(err)
	}
	ids := make([]string, len(participants))
	for i, p := range participants {
		ids[i] = p.UserID.String()
	}
	return &session_manager_service.ParticipantsResponse{
		ParticipantIds: ids,
	}, nil
}

func (s *Server) JoinSession(ctx context.Context, req *session_manager_service.JoinSessionRequest) (*session_manager_service.SessionResponse, error) {
	// Сначала проверяем наличие обязательных полей
	if req.GetPin() == "" && req.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id or pin required")
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	var ses *model.ConsultationSession
	if req.GetPin() != "" {
		ses, err = s.Session.JoinByPIN(req.GetPin(), userID)
		if err != nil {
			return nil, s.mapError(err)
		}
		if ses == nil {
			return nil, status.Error(codes.NotFound, "session not found")
		}
	} else if req.GetSessionId() != "" {
		sessionID, parseErr := uuid.Parse(req.GetSessionId())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid session_id")
		}
		ses, err = s.Session.JoinBySessionID(sessionID, userID)
		if err != nil {
			return nil, s.mapError(err)
		}
		if ses == nil {
			return nil, status.Error(codes.NotFound, "session not found")
		}
	} else {
		// Это не должно произойти из-за проверки выше, но на всякий случай
		return nil, status.Error(codes.InvalidArgument, "session_id or pin required")
	}
	// Индексация теперь через Kafka consumer в search-service worker
	// Fire-and-forget: событие должно уйти даже при отмене запроса, но с таймаутом
	if s.Producer != nil {
		eventCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		go s.Producer.ProduceSessionEvent(eventCtx, "operator_joined", ses.ID, map[string]interface{}{
			"user_id":   userID.String(),
			"client_id": ses.ClientID.String(),
			"pin":       ses.PIN,
			"status":    ses.Status,
		})
	}
	return toProtoSession(ses), nil
}

func (s *Server) Invite(ctx context.Context, req *session_manager_service.InviteRequest) (*session_manager_service.InviteResponse, error) {
	sessionID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	operatorID, err := uuid.Parse(req.GetOperatorId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid operator_id")
	}
	if err := s.Session.Invite(sessionID, operatorID); err != nil {
		return nil, s.mapError(err)
	}
	// Индексация теперь через Kafka consumer в search-service worker
	// Fire-and-forget: событие должно уйти даже при отмене запроса, но с таймаутом
	if s.Producer != nil {
		// Получаем сессию для отправки полных данных
		if ses, err := s.Session.GetByID(sessionID); err == nil && ses != nil {
			eventCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			go s.Producer.ProduceSessionEvent(eventCtx, "operator_joined", sessionID, map[string]interface{}{
				"operator_id": operatorID.String(),
				"client_id":   ses.ClientID.String(),
				"pin":         ses.PIN,
				"status":      ses.Status,
			})
		}
	}
	return &session_manager_service.InviteResponse{Ok: true}, nil
}

func (s *Server) Control(ctx context.Context, req *session_manager_service.ControlRequest) (*session_manager_service.ControlResponse, error) {
	sessionID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	// Validate action before permission check so missing action returns 400.
	statusStr := req.GetAction()
	if statusStr == "" {
		return nil, status.Error(codes.InvalidArgument, "action is required")
	}
	// Permission check: caller must be a session participant (client or operator).
	callerIDStr := getMetadata(ctx, "x-caller-id")
	if callerIDStr == "" {
		return nil, status.Error(codes.PermissionDenied, "caller identity required (x-caller-id)")
	}
	callerID, err := uuid.Parse(callerIDStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid x-caller-id")
	}
	ok, err := s.Session.IsParticipant(sessionID, callerID)
	if err != nil {
		return nil, s.mapError(err)
	}
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "caller is not a participant of this session")
	}
	var leadOperatorID *uuid.UUID
	if err := s.Session.Control(sessionID, leadOperatorID, statusStr); err != nil {
		return nil, s.mapError(err)
	}
	// Индексация теперь через Kafka consumer в search-service worker
	// Fire-and-forget: событие должно уйти даже при отмене запроса, но с таймаутом
	if s.Producer != nil {
		// Получаем сессию для отправки полных данных
		if ses, err := s.Session.GetByID(sessionID); err == nil && ses != nil {
			eventType := "session.updated"
			if statusStr == "finished" {
				eventType = "session.ended"
			}
			eventCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			go s.Producer.ProduceSessionEvent(eventCtx, eventType, sessionID, map[string]interface{}{
				"client_id": ses.ClientID.String(),
				"pin":       ses.PIN,
				"status":    ses.Status, // Обновлённый статус
			})
		}
	}
	return &session_manager_service.ControlResponse{Ok: true}, nil
}

func (s *Server) SetRecordingUrl(ctx context.Context, req *session_manager_service.SetRecordingUrlRequest) (*session_manager_service.SetRecordingUrlResponse, error) {
	if req.GetStreamSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "stream_session_id is required")
	}
	streamSessionID, err := uuid.Parse(req.GetStreamSessionId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid stream_session_id")
	}
	if err := s.Session.SetRecordingUrlByStreamSessionID(streamSessionID, req.GetRecordingUrl()); err != nil {
		return nil, s.mapError(err)
	}
	return &session_manager_service.SetRecordingUrlResponse{Ok: true}, nil
}
