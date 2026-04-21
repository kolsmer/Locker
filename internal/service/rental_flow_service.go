package service

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"locker/internal/repository"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *AppError) Error() string {
	return e.Code
}

type RentalFlowService struct {
	repo *repository.RentalFlowRepository
}

func NewRentalFlowService(repo *repository.RentalFlowRepository) *RentalFlowService {
	return &RentalFlowService{repo: repo}
}

func (s *RentalFlowService) Init(ctx context.Context) error {
	return s.repo.EnsureDemoData(ctx)
}

func (s *RentalFlowService) CleanupExpiredSelections(ctx context.Context) error {
	return s.repo.CleanupExpiredSelections(ctx)
}

func (s *RentalFlowService) GetLockers(ctx context.Context, city string, limitStr string, offsetStr string) ([]map[string]interface{}, int, error) {
	city = strings.ToLower(strings.TrimSpace(city))
	limit := 100
	offset := 0

	if limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v < 0 {
			return nil, 0, &AppError{Status: http.StatusUnprocessableEntity, Code: "INVALID_LIMIT", Message: "Некорректный limit", Details: map[string]interface{}{"field": "limit"}}
		}
		limit = v
	}
	if offsetStr != "" {
		v, err := strconv.Atoi(offsetStr)
		if err != nil || v < 0 {
			return nil, 0, &AppError{Status: http.StatusUnprocessableEntity, Code: "INVALID_OFFSET", Message: "Некорректный offset", Details: map[string]interface{}{"field": "offset"}}
		}
		offset = v
	}

	return s.repo.ListLockers(ctx, city, limit, offset)
}

func (s *RentalFlowService) CreateCellSelection(ctx context.Context, lockerID int64, size string, dimensions map[string]interface{}) (map[string]interface{}, error) {
	size = strings.ToLower(strings.TrimSpace(size))

	if size == "" && dimensions != nil {
		length, lok := dimensions["length"].(float64)
		width, wok := dimensions["width"].(float64)
		height, hok := dimensions["height"].(float64)
		unit, _ := dimensions["unit"].(string)
		if !lok || !wok || !hok || length <= 0 || width <= 0 || height <= 0 {
			return nil, &AppError{Status: http.StatusUnprocessableEntity, Code: "INVALID_DIMENSIONS", Message: "Некорректные габариты", Details: map[string]interface{}{"field": "dimensions"}}
		}
		if unit != "" && strings.ToLower(unit) != "cm" {
			return nil, &AppError{Status: http.StatusUnprocessableEntity, Code: "INVALID_DIMENSIONS", Message: "Поддерживается только unit=cm", Details: map[string]interface{}{"field": "dimensions.unit"}}
		}
		mapped, ok := sizeFromDimensions(length, width, height)
		if !ok {
			return nil, &AppError{Status: http.StatusUnprocessableEntity, Code: "INVALID_DIMENSIONS", Message: "Габариты не поддерживаются", Details: map[string]interface{}{"field": "dimensions"}}
		}
		size = mapped
	}

	if !validSize(size) {
		return nil, &AppError{Status: http.StatusUnprocessableEntity, Code: "INVALID_SIZE", Message: "Некорректный размер ячейки", Details: map[string]interface{}{"field": "size"}}
	}

	result, err := s.repo.CreateCellSelection(ctx, lockerID, size)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return result, nil
}

func (s *RentalFlowService) CreateBooking(ctx context.Context, lockerID int64, selectionID string, phone string) (map[string]interface{}, error) {
	normPhone, ok := normalizePhone(phone)
	if !ok {
		return nil, &AppError{Status: http.StatusUnprocessableEntity, Code: "INVALID_PHONE", Message: "Введите корректный номер телефона", Details: map[string]interface{}{"field": "phone"}}
	}

	result, err := s.repo.CreateBooking(ctx, lockerID, selectionID, normPhone)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return result, nil
}

func (s *RentalFlowService) CheckAccessCode(ctx context.Context, lockerID int64, accessCode string) (map[string]interface{}, error) {
	code := strings.ToUpper(strings.TrimSpace(accessCode))
	if len(code) != 6 {
		return nil, &AppError{Status: http.StatusNotFound, Code: "INVALID_ACCESS_CODE", Message: "Неверный код доступа"}
	}
	result, err := s.repo.CheckAccessCode(ctx, lockerID, code)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return result, nil
}

func (s *RentalFlowService) GetPayment(ctx context.Context, paymentID string) (map[string]interface{}, error) {
	result, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return result, nil
}

func (s *RentalFlowService) OpenRental(ctx context.Context, rentalID string) (map[string]interface{}, error) {
	result, err := s.repo.OpenRental(ctx, rentalID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return result, nil
}

func (s *RentalFlowService) FinishRental(ctx context.Context, rentalID string) (map[string]interface{}, error) {
	result, err := s.repo.FinishRental(ctx, rentalID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return result, nil
}

func (s *RentalFlowService) GetRental(ctx context.Context, rentalID string) (map[string]interface{}, error) {
	result, err := s.repo.GetRental(ctx, rentalID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return result, nil
}

func mapRepoErr(err error) error {
	switch {
	case errors.Is(err, repository.ErrLockerNotFound):
		return &AppError{Status: http.StatusNotFound, Code: "LOCKER_NOT_FOUND", Message: "Локер не найден"}
	case errors.Is(err, repository.ErrNoCellsAvailable):
		return &AppError{Status: http.StatusConflict, Code: "NO_CELLS_AVAILABLE", Message: "Свободных ячеек этого размера нет"}
	case errors.Is(err, repository.ErrSelectionExpired):
		return &AppError{Status: http.StatusGone, Code: "SELECTION_EXPIRED", Message: "Бронь истекла"}
	case errors.Is(err, repository.ErrInvalidAccess):
		return &AppError{Status: http.StatusNotFound, Code: "INVALID_ACCESS_CODE", Message: "Неверный код доступа"}
	case errors.Is(err, repository.ErrPaymentRequired):
		return &AppError{Status: http.StatusPaymentRequired, Code: "PAYMENT_REQUIRED", Message: "Требуется оплата"}
	case errors.Is(err, repository.ErrPaymentNotFound):
		return &AppError{Status: http.StatusNotFound, Code: "PAYMENT_NOT_FOUND", Message: "Платеж не найден"}
	case errors.Is(err, repository.ErrRentalClosed):
		return &AppError{Status: http.StatusConflict, Code: "RENTAL_CLOSED", Message: "Аренда уже завершена"}
	case errors.Is(err, repository.ErrRentalNotFound):
		return &AppError{Status: http.StatusNotFound, Code: "RENTAL_NOT_FOUND", Message: "Аренда не найдена"}
	default:
		return err
	}
}

func normalizePhone(phone string) (string, bool) {
	digits := make([]rune, 0, len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) != 11 {
		return "", false
	}
	if digits[0] == '8' {
		digits[0] = '7'
	}
	if digits[0] != '7' {
		return "", false
	}
	return "+" + string(digits), true
}

func validSize(size string) bool {
	s := strings.ToLower(strings.TrimSpace(size))
	return s == "s" || s == "m" || s == "l" || s == "xl"
}

func sizeFromDimensions(length, width, height float64) (string, bool) {
	maxD := length
	if width > maxD {
		maxD = width
	}
	if height > maxD {
		maxD = height
	}
	if maxD <= 25 {
		return "s", true
	}
	if maxD <= 45 {
		return "m", true
	}
	if maxD <= 65 {
		return "l", true
	}
	if maxD <= 90 {
		return "xl", true
	}
	return "", false
}
