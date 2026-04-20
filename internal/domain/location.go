package domain

// Location локация (шкаф, постомат)
type Location struct {
	ID        int64
	Name      string
	Address   string
	Latitude  float64
	Longitude float64
	IsActive  bool
	CreatedAt int64
	UpdatedAt int64
}
