package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"locker/internal/domain"
	"locker/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

type AdminService struct {
	adminRepo *repository.AdminRepository
	panelRepo *repository.AdminPanelRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewAdminService(adminRepo *repository.AdminRepository, panelRepo *repository.AdminPanelRepository, jwtSecret string, tokenTTL time.Duration) *AdminService {
	if tokenTTL <= 0 {
		tokenTTL = time.Hour
	}
	return &AdminService{adminRepo: adminRepo, panelRepo: panelRepo, jwtSecret: []byte(jwtSecret), tokenTTL: tokenTTL}
}

func (s *AdminService) Login(ctx context.Context, login string, password string) (map[string]interface{}, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return nil, &AppError{Status: 401, Code: "INVALID_CREDENTIALS", Message: "Неверный логин или пароль"}
	}

	admin, err := s.adminRepo.GetByLogin(ctx, login)
	if err != nil {
		return nil, &AppError{Status: 401, Code: "INVALID_CREDENTIALS", Message: "Неверный логин или пароль"}
	}
	if !admin.IsActive {
		return nil, &AppError{Status: 403, Code: "ADMIN_DISABLED", Message: "Админ отключен"}
	}
	if !checkPassword(admin.PasswordHash, password) {
		return nil, &AppError{Status: 401, Code: "INVALID_CREDENTIALS", Message: "Неверный логин или пароль"}
	}

	expiresAt := time.Now().Add(s.tokenTTL)
	token, err := s.generateToken(admin, expiresAt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"accessToken": token,
		"tokenType":   "Bearer",
		"expiresIn":   int(s.tokenTTL.Seconds()),
		"admin": map[string]interface{}{
			"id":    admin.ID,
			"login": admin.Login,
			"role":  string(admin.Role),
		},
	}, nil
}

func (s *AdminService) Me(ctx context.Context, adminID int64) (map[string]interface{}, error) {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil {
		return nil, &AppError{Status: 401, Code: "UNAUTHORIZED", Message: "Требуется авторизация"}
	}

	return map[string]interface{}{
		"id":       admin.ID,
		"login":    admin.Login,
		"role":     string(admin.Role),
		"isActive": admin.IsActive,
	}, nil
}

func (s *AdminService) ListLocations(ctx context.Context, search string, isActiveRaw string, limitRaw string, offsetRaw string) ([]map[string]interface{}, int, error) {
	limit, offset, err := parseLimitOffset(limitRaw, offsetRaw)
	if err != nil {
		return nil, 0, err
	}

	isActive, err := parseOptionalBool(isActiveRaw)
	if err != nil {
		return nil, 0, &AppError{Status: 422, Code: "INVALID_FILTER", Message: "Некорректный isActive"}
	}

	return s.panelRepo.ListLocations(ctx, strings.ToLower(strings.TrimSpace(search)), isActive, limit, offset)
}

func (s *AdminService) ListLocationLockers(ctx context.Context, locationID int64, statusRaw string, sizeRaw string, isActiveRaw string, limitRaw string, offsetRaw string) ([]map[string]interface{}, int, error) {
	limit, offset, err := parseLimitOffset(limitRaw, offsetRaw)
	if err != nil {
		return nil, 0, err
	}
	isActive, err := parseOptionalBool(isActiveRaw)
	if err != nil {
		return nil, 0, &AppError{Status: 422, Code: "INVALID_FILTER", Message: "Некорректный isActive"}
	}
	statuses := splitCSVLower(statusRaw)
	sizes := splitCSVUpper(sizeRaw)
	return s.panelRepo.ListLocationLockers(ctx, locationID, statuses, sizes, isActive, limit, offset)
}

func (s *AdminService) LockerDetail(ctx context.Context, lockerID int64) (map[string]interface{}, error) {
	data, err := s.panelRepo.GetLockerDetail(ctx, lockerID)
	if err != nil {
		if errors.Is(err, repository.ErrLockerNotFound) {
			return nil, &AppError{Status: 404, Code: "LOCKER_NOT_FOUND", Message: "Ячейка не найдена"}
		}
		return nil, err
	}
	return data, nil
}

func (s *AdminService) UpdateLockerStatus(ctx context.Context, adminID int64, role string, lockerID int64, newStatus string, reason string) (map[string]interface{}, error) {
	if !canManage(role) {
		return nil, &AppError{Status: 403, Code: "FORBIDDEN", Message: "Недостаточно прав"}
	}
	newStatus = strings.TrimSpace(strings.ToLower(newStatus))
	if !validLockerStatus(newStatus) {
		return nil, &AppError{Status: 422, Code: "INVALID_STATUS", Message: "Некорректный статус"}
	}
	prev, updatedAt, err := s.panelRepo.UpdateLockerStatus(ctx, lockerID, newStatus, strings.TrimSpace(reason), adminID)
	if err != nil {
		if errors.Is(err, repository.ErrLockerNotFound) {
			return nil, &AppError{Status: 404, Code: "LOCKER_NOT_FOUND", Message: "Ячейка не найдена"}
		}
		if errors.Is(err, repository.ErrPaymentRequired) {
			return nil, &AppError{Status: 409, Code: "ACTIVE_RENTAL_EXISTS", Message: "Есть активная аренда"}
		}
		return nil, err
	}
	return map[string]interface{}{
		"lockerId":       lockerID,
		"previousStatus": prev,
		"newStatus":      newStatus,
		"updatedAt":      time.Unix(updatedAt, 0).UTC().Format(time.RFC3339),
	}, nil
}

func (s *AdminService) ManualOpenLocker(ctx context.Context, adminID int64, role string, lockerID int64, reason string) (map[string]interface{}, error) {
	if !canManage(role) {
		return nil, &AppError{Status: 403, Code: "FORBIDDEN", Message: "Недостаточно прав"}
	}
	commandID, err := s.panelRepo.ManualOpenLocker(ctx, lockerID, strings.TrimSpace(reason), adminID)
	if err != nil {
		if errors.Is(err, repository.ErrLockerNotFound) {
			return nil, &AppError{Status: 404, Code: "LOCKER_NOT_FOUND", Message: "Ячейка не найдена"}
		}
		if errors.Is(err, repository.ErrNoCellsAvailable) {
			return nil, &AppError{Status: 409, Code: "LOCKER_NOT_FUNCTIONAL", Message: "Ячейка недоступна"}
		}
		return nil, err
	}
	return map[string]interface{}{
		"lockerId":  lockerID,
		"commandId": commandID,
		"status":    "pending",
	}, nil
}

func (s *AdminService) ListSessions(ctx context.Context, locationIDRaw string, lockerIDRaw string, statusRaw string, phone string, fromRaw string, toRaw string, limitRaw string, offsetRaw string) ([]map[string]interface{}, int, error) {
	limit, offset, err := parseLimitOffset(limitRaw, offsetRaw)
	if err != nil {
		return nil, 0, err
	}
	locationID, err := parseOptionalInt64(locationIDRaw)
	if err != nil {
		return nil, 0, &AppError{Status: 422, Code: "INVALID_FILTER", Message: "Некорректный locationId"}
	}
	lockerID, err := parseOptionalInt64(lockerIDRaw)
	if err != nil {
		return nil, 0, &AppError{Status: 422, Code: "INVALID_FILTER", Message: "Некорректный lockerId"}
	}
	from, err := parseOptionalUnixOrDate(fromRaw)
	if err != nil {
		return nil, 0, &AppError{Status: 422, Code: "INVALID_FILTER", Message: "Некорректный from"}
	}
	to, err := parseOptionalUnixOrDate(toRaw)
	if err != nil {
		return nil, 0, &AppError{Status: 422, Code: "INVALID_FILTER", Message: "Некорректный to"}
	}
	statuses := splitCSVLower(statusRaw)
	return s.panelRepo.ListSessions(ctx, locationID, lockerID, statuses, strings.TrimSpace(phone), from, to, limit, offset)
}

func (s *AdminService) RevenueExport(ctx context.Context, role string, fromRaw string, toRaw string, locationIDRaw string) ([]byte, string, error) {
	if role != string(domain.AdminRoleAdmin) {
		return nil, "", &AppError{Status: 403, Code: "FORBIDDEN", Message: "Недостаточно прав"}
	}
	fromDate, err := time.Parse("2006-01-02", strings.TrimSpace(fromRaw))
	if err != nil {
		return nil, "", &AppError{Status: 422, Code: "INVALID_DATE_RANGE", Message: "Некорректный from"}
	}
	toDate, err := time.Parse("2006-01-02", strings.TrimSpace(toRaw))
	if err != nil {
		return nil, "", &AppError{Status: 422, Code: "INVALID_DATE_RANGE", Message: "Некорректный to"}
	}
	if toDate.Before(fromDate) {
		return nil, "", &AppError{Status: 422, Code: "INVALID_DATE_RANGE", Message: "Некорректный диапазон дат"}
	}
	toExclusive := toDate.Add(24 * time.Hour)

	locationID, err := parseOptionalInt64(locationIDRaw)
	if err != nil {
		return nil, "", &AppError{Status: 422, Code: "INVALID_FILTER", Message: "Некорректный locationId"}
	}

	rows, err := s.panelRepo.RevenueByLocation(ctx, fromDate.UTC(), toExclusive.UTC(), locationID)
	if err != nil {
		return nil, "", &AppError{Status: 500, Code: "EXPORT_GENERATION_FAILED", Message: "Ошибка генерации файла"}
	}

	f := excelize.NewFile()
	sheet := "Revenue"
	f.SetSheetName("Sheet1", sheet)
	headers := []string{"location_id", "location_name", "address", "payments_count", "revenue_rub", "avg_check_rub", "first_payment_at", "last_payment_at"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, row := range rows {
		n := i + 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", n), row.LocationID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", n), row.LocationName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", n), row.Address)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", n), row.PaymentsCount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", n), row.RevenueRUB)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", n), row.AvgCheckRUB)
		if row.FirstPaymentAt != nil {
			_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", n), row.FirstPaymentAt.UTC().Format(time.RFC3339))
		}
		if row.LastPaymentAt != nil {
			_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", n), row.LastPaymentAt.UTC().Format(time.RFC3339))
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", &AppError{Status: 500, Code: "EXPORT_GENERATION_FAILED", Message: "Ошибка генерации файла"}
	}
	filename := fmt.Sprintf("revenue_%s_%s.xlsx", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
	return buf.Bytes(), filename, nil
}

func (s *AdminService) generateToken(admin *domain.Admin, expiresAt time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":   strconv.FormatInt(admin.ID, 10),
		"login": admin.Login,
		"role":  string(admin.Role),
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func checkPassword(hash string, password string) bool {
	if strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") || strings.HasPrefix(hash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	return hash == password
}

func canManage(role string) bool {
	return role == string(domain.AdminRoleAdmin) || role == string(domain.AdminRoleOperator)
}

func validLockerStatus(s string) bool {
	switch s {
	case "free", "reserved", "occupied", "locked", "open", "maintenance", "out_of_service":
		return true
	default:
		return false
	}
}

func parseLimitOffset(limitRaw string, offsetRaw string) (int, int, error) {
	limit := 50
	offset := 0
	if strings.TrimSpace(limitRaw) != "" {
		v, err := strconv.Atoi(limitRaw)
		if err != nil || v < 0 {
			return 0, 0, &AppError{Status: 422, Code: "INVALID_LIMIT", Message: "Некорректный limit"}
		}
		limit = v
	}
	if strings.TrimSpace(offsetRaw) != "" {
		v, err := strconv.Atoi(offsetRaw)
		if err != nil || v < 0 {
			return 0, 0, &AppError{Status: 422, Code: "INVALID_OFFSET", Message: "Некорректный offset"}
		}
		offset = v
	}
	return limit, offset, nil
}

func parseOptionalBool(raw string) (*bool, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil, nil
	}
	if raw == "true" || raw == "1" {
		v := true
		return &v, nil
	}
	if raw == "false" || raw == "0" {
		v := false
		return &v, nil
	}
	return nil, errors.New("invalid bool")
}

func parseOptionalInt64(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseOptionalUnixOrDate(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return &n, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		n := t.Unix()
		return &n, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		n := t.Unix()
		return &n, nil
	}
	return nil, errors.New("invalid time")
}

func splitCSVLower(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func splitCSVUpper(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.ToUpper(strings.TrimSpace(p))
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
