package domain

// DeviceCommandType тип команды на устройство
type DeviceCommandType string

const (
	CmdOpenLock  DeviceCommandType = "open_lock"
	CmdCloseLock DeviceCommandType = "close_lock"
)

// DeviceCommandStatus статус команды
type DeviceCommandStatus string

const (
	CmdStatusPending   DeviceCommandStatus = "pending"
	CmdStatusExecuted  DeviceCommandStatus = "executed"
	CmdStatusFailed    DeviceCommandStatus = "failed"
)

// DeviceCommand команда для постомата
type DeviceCommand struct {
	ID        int64
	DeviceID  string
	LockerID  int64
	SessionID int64
	Type      DeviceCommandType
	Status    DeviceCommandStatus
	Retries   int
	Error     string
	CreatedAt int64
	FetchedAt int64
	DoneAt    int64
}

// DeviceEvent событие от устройства
type DeviceEvent struct {
	ID        int64
	DeviceID  string
	LockerID  int64
	EventType string
	Payload   string
	CreatedAt int64
}
