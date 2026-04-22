package domain

// PaymentStatus статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusConfirmed PaymentStatus = "confirmed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

const PricePerMinute = 10.0

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

func CalculatePaymentAmount(startTime, endTime int64) float64 {
	if endTime <= startTime {
		return 0
	}
	
	durationSeconds := float64(endTime - startTime)
	durationMinutes := durationSeconds / 60.0
	
	minutes := int64((durationMinutes + 0.5))
	if minutes < 1 {
		minutes = 1
	}
	
	return float64(minutes) * PricePerMinute
}
