package grpc

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/psds-microservice/session-manager-service/internal/errs"
	"github.com/psds-microservice/session-manager-service/internal/model"
	"github.com/psds-microservice/session-manager-service/internal/service"
	"github.com/psds-microservice/session-manager-service/pkg/gen/session_manager_service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Deps — зависимости gRPC-сервера (D: зависимость от абстракций).
type Deps struct {
	Session service.SessionServicer
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
		Id:     ses.ID.String(),
		Status: ses.Status,
	}
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
	return toProtoSession(ses), nil
}

func (s *Server) GetParticipants(ctx context.Context, req *session_manager_service.GetParticipantsRequest) (*session_manager_service.ParticipantsResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
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
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	var ses *model.ConsultationSession
	if req.GetPin() != "" {
		ses, err = s.Session.JoinByPIN(req.GetPin(), userID)
	} else if req.GetSessionId() != "" {
		sessionID, err := uuid.Parse(req.GetSessionId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid session_id")
		}
		ses, err = s.Session.JoinBySessionID(sessionID, userID)
	} else {
		return nil, status.Error(codes.InvalidArgument, "session_id or pin required")
	}
	if err != nil {
		return nil, s.mapError(err)
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
	return &session_manager_service.InviteResponse{Ok: true}, nil
}

func (s *Server) Control(ctx context.Context, req *session_manager_service.ControlRequest) (*session_manager_service.ControlResponse, error) {
	sessionID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	// В proto ControlRequest имеет только id и action
	// Используем action как status (active, finished)
	var leadOperatorID *uuid.UUID
	statusStr := req.GetAction()
	if err := s.Session.Control(sessionID, leadOperatorID, statusStr); err != nil {
		return nil, s.mapError(err)
	}
	return &session_manager_service.ControlResponse{Ok: true}, nil
}
