package domain

// AdminRole роль админа
type AdminRole string

const (
	AdminRoleAdmin    AdminRole = "admin"
	AdminRoleOperator AdminRole = "operator"
	AdminRoleSupport  AdminRole = "support"
)

// Admin администратор
type Admin struct {
	ID           int64
	Login        string
	PasswordHash string
	Role         AdminRole
	IsActive     bool
	CreatedAt    int64
	UpdatedAt    int64
}

func (a *Admin) CanManualOpen() bool {
	return a.Role == AdminRoleAdmin || a.Role == AdminRoleOperator
}

func (a *Admin) CanViewRevenue() bool {
	return a.Role == AdminRoleAdmin
}
