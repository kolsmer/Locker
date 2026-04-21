package domain

// EventType тип события
type EventType string

const (
	EventTypeLockerOpened      EventType = "locker_opened"
	EventTypeLockerClosed      EventType = "locker_closed"
	EventTypeSessionCreated    EventType = "session_created"
	EventTypeSessionPaid       EventType = "session_paid"
	EventTypeSessionClosed     EventType = "session_closed"
	EventTypeAccessCodeUsed    EventType = "access_code_used"
	EventTypeAccessCodeFailed  EventType = "access_code_failed"
	EventTypePaymentReceived   EventType = "payment_received"
	EventTypePaymentFailed     EventType = "payment_failed"
	EventTypeHardwareError     EventType = "hardware_error"
	EventTypeAdminAction       EventType = "admin_action"
)

// Event событие
type Event struct {
	ID        int64
	LockerID  int64
	SessionID int64
	Type      EventType
	Payload   string
	CreatedAt int64
}

// AuditLog аудит всех админ-действий
type AuditLog struct {
	ID         int64
	ActorType  string // "admin", "system"
	ActorID    int64
	Action     string
	ObjectType string // "locker", "session", "payment"
	ObjectID   int64
	Payload    string
	CreatedAt  int64
}
