package repository

import (
	"context"
	"database/sql"
	"locker/internal/domain"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetByLogin(ctx context.Context, login string) (*domain.Admin, error) {
	query := `SELECT id, login, password_hash, role, is_active, created_at, updated_at FROM admins WHERE login = $1`
	row := r.db.QueryRowContext(ctx, query, login)
	admin := &domain.Admin{}
	err := row.Scan(&admin.ID, &admin.Login, &admin.PasswordHash, &admin.Role, &admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return admin, nil
}

func (r *AdminRepository) GetByID(ctx context.Context, id int64) (*domain.Admin, error) {
	query := `SELECT id, login, password_hash, role, is_active, created_at, updated_at FROM admins WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	admin := &domain.Admin{}
	err := row.Scan(&admin.ID, &admin.Login, &admin.PasswordHash, &admin.Role, &admin.IsActive, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return admin, nil
}

func (r *AdminRepository) Create(ctx context.Context, admin *domain.Admin) (int64, error) {
	query := `
		INSERT INTO admins (login, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, admin.Login, admin.PasswordHash, admin.Role, admin.IsActive, nowUnix(), nowUnix()).Scan(&id)
	return id, err
}
