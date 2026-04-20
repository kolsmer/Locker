package service

import (
	"context"
	"errors"
	"locker/internal/domain"
	"locker/internal/repository"
)

type PaymentService struct {
	paymentRepo *repository.PaymentRepository
	sessionRepo *repository.SessionRepository
	eventRepo   *repository.EventRepository
}

func NewPaymentService(
	paymentRepo *repository.PaymentRepository,
	sessionRepo *repository.SessionRepository,
	eventRepo *repository.EventRepository,
) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		sessionRepo: sessionRepo,
		eventRepo:   eventRepo,
	}
}

func (s *PaymentService) CreatePaymentIntent(ctx context.Context, sessionID int64, amount float64) (*domain.Payment, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// TODO: check session is not already paid
	if session.Status == domain.SessionStatusPaid {
		return nil, errors.New("session already paid")
	}

	payment := &domain.Payment{
		SessionID: sessionID,
		Amount:    amount,
		Currency:  "RUB",
		Status:    domain.PaymentStatusPending,
		Provider:  "yookassa", // TODO: configurable
	}

	id, err := s.paymentRepo.Create(ctx, payment)
	if err != nil {
		return nil, err
	}

	payment.ID = id

	// TODO: generate QR code with payment link
	payment.QRPayload = "https://example.com/pay/" // placeholder

	return payment, nil
}

func (s *PaymentService) HandlePaymentCallback(ctx context.Context, externalPaymentID string, status domain.PaymentStatus) error {
	// TODO: find payment by external ID
	// TODO: verify signature
	// TODO: update payment status
	return nil
}
