package domain

// LockerStatus состояние ячейки
type LockerStatus string

const (
	LockerStatusFree         LockerStatus = "free"
	LockerStatusReserved     LockerStatus = "reserved"
	LockerStatusOccupied     LockerStatus = "occupied"
	LockerStatusLocked       LockerStatus = "locked"
	LockerStatusOpen         LockerStatus = "open"
	LockerStatusMaintenance  LockerStatus = "maintenance"
	LockerStatusOutOfService LockerStatus = "out_of_service"
)

// LockerSize размер ячейки
type LockerSize string

const (
	LockerSizeS  LockerSize = "S"
	LockerSizeM  LockerSize = "M"
	LockerSizeL  LockerSize = "L"
	LockerSizeXL LockerSize = "XL"
)

// Locker как физическая ячейка
type Locker struct {
	ID          int64
	LocationID  int64
	LockerNo    int
	Size        LockerSize
	Status      LockerStatus
	HardwareID  string
	IsActive    bool
	Price       float64
	CreatedAt   int64
	UpdatedAt   int64
	LastEventAt int64
}

func (l *Locker) CanOpenForSession() bool {
	return l.Status == LockerStatusFree && l.IsActive
}

func (l *Locker) IsFunctional() bool {
	return l.IsActive && l.Status != LockerStatusOutOfService && l.Status != LockerStatusMaintenance
}
