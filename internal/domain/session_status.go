package domain

// SessionStatus статус хранения
type SessionStatus string

const (
	SessionStatusCreated    SessionStatus = "created"
	SessionStatusWaitingPay SessionStatus = "waiting_payment"
	SessionStatusPaid       SessionStatus = "paid"
	SessionStatusActive     SessionStatus = "active"      // вещи внутри
	SessionStatusClosed     SessionStatus = "closed"      // забрал вещи
	SessionStatusExpired    SessionStatus = "expired"     // время истекло
	SessionStatusCancelled  SessionStatus = "cancelled"
	SessionStatusError      SessionStatus = "error"
)

// StorageSession главная бизнес сущность
type StorageSession struct {
	ID            int64
	LockerID      int64
	Phone        string
	AccessCode    string
	Status        SessionStatus
	StartedAt     int64
	EndsAt        int64
	PaidUntil     int64
	ClosedAt      int64
	OpenAttempts  int
	CreatedSource string // "client", "postomat", "admin"
	CreatedAt     int64
	UpdatedAt     int64
}

func (s *StorageSession) IsActive() bool {
	return s.Status == SessionStatusActive || s.Status == SessionStatusPaid
}

func (s *StorageSession) CanUseAccessCode() bool {
	return s.Status == SessionStatusPaid && s.PaidUntil > timeNow()
}

func timeNow() int64 {
	// return time.Now().Unix()
	return 0
}
