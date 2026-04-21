package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"locker/internal/domain"
	"locker/internal/repository"
)

type SessionService struct {
	sessionRepo *repository.SessionRepository
	lockerRepo  *repository.LockerRepository
	eventRepo   *repository.EventRepository
}

func NewSessionService(
	sessionRepo *repository.SessionRepository,
	lockerRepo *repository.LockerRepository,
	eventRepo *repository.EventRepository,
) *SessionService {
	return &SessionService{
		sessionRepo: sessionRepo,
		lockerRepo:  lockerRepo,
		eventRepo:   eventRepo,
	}
}

func (s *SessionService) CreateSession(ctx context.Context, lockerID int64, phone string, source string) (*domain.StorageSession, error) {
	locker, err := s.lockerRepo.GetByID(ctx, lockerID)
	if err != nil {
		return nil, err
	}

	if !locker.CanOpenForSession() {
		return nil, errors.New("locker not available")
	}

	session := &domain.StorageSession{
		LockerID:      lockerID,
		Phone:        phone,
		Status:        domain.SessionStatusCreated,
		StartedAt:     nowUnix(),
		CreatedSource: source,
	}

	id, err := s.sessionRepo.Create(ctx, session)
	if err != nil {
		return nil, err
	}

	session.ID = id
	s.eventRepo.LogLockerEvent(ctx, &domain.Event{
		LockerID:  lockerID,
		SessionID: id,
		Type:      domain.EventTypeSessionCreated,
		Payload:   fmt.Sprintf(`{"phone":"%s"}`, phone),
		CreatedAt: nowUnix(),
	})

	return session, nil
}

func (s *SessionService) MarkPaid(ctx context.Context, sessionID int64, paidUntil int64) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != domain.SessionStatusWaitingPay && session.Status != domain.SessionStatusCreated {
		return errors.New("invalid session status for payment")
	}

	err = s.sessionRepo.UpdateStatus(ctx, sessionID, domain.SessionStatusPaid)
	if err != nil {
		return err
	}

	err = s.sessionRepo.UpdatePaidUntil(ctx, sessionID, paidUntil)
	if err != nil {
		return err
	}

	s.eventRepo.LogLockerEvent(ctx, &domain.Event{
		LockerID:  session.LockerID,
		SessionID: sessionID,
		Type:      domain.EventTypeSessionPaid,
		CreatedAt: nowUnix(),
	})

	return nil
}

func (s *SessionService) VerifyAccessCode(ctx context.Context, sessionID int64, accessCode string) (bool, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return false, err
	}

	if !session.CanUseAccessCode() {
		return false, errors.New("code expired or session not paid")
	}

	hash := hashCode(accessCode)
	if hash != session.AccessCode {
		s.sessionRepo.IncrementOpenAttempts(ctx, sessionID)
		return false, nil
	}

	return true, nil
}

func (s *SessionService) GenAccessCode(ctx context.Context, sessionID int64) (string, error) {
	code := generateCode()
	hash := hashCode(code)

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return "", err
	}

	session.AccessCode = hash
	// TODO: save to db

	return code, nil
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

func generateCode() string {
	return "000000" // TODO: random 6 digits
}

func nowUnix() int64 {
	return 0 // TODO: time.Now().Unix()
}
