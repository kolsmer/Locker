package domain

// PaymentStatus статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusConfirmed PaymentStatus = "confirmed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// Payment платёж
type Payment struct {
	ID                int64
	SessionID         int64
	Amount            float64
	Currency          string
	Status            PaymentStatus
	ExternalPaymentID string
	Provider          string
	QRPayload         string
	PaidAt            int64
	CreatedAt         int64
	UpdatedAt         int64
	RawCallbackJSON   string
}

func (p *Payment) IsPaid() bool {
	return p.Status == PaymentStatusConfirmed
}
