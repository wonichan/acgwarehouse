package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/ports"
	"github.com/yachiyo/acgwarehouse/internal/service"
)

type memoryCheckInRepository struct {
	nextID  int64
	records map[string]do.CheckIn
}

func newMemoryCheckInRepository() *memoryCheckInRepository {
	return &memoryCheckInRepository{
		nextID:  1,
		records: make(map[string]do.CheckIn),
	}
}

func checkInKey(userID int64, date string) string {
	return fmt.Sprintf("%d|%s", userID, date)
}

func (r *memoryCheckInRepository) CheckInToday(_ context.Context, userID int64, date string, pointsAwarded int) (bool, error) {
	k := checkInKey(userID, date)
	if _, exists := r.records[k]; exists {
		return false, nil
	}
	r.records[k] = do.CheckIn{
		ID:            r.nextID,
		UserID:        userID,
		CheckInDate:   date,
		PointsAwarded: pointsAwarded,
	}
	r.nextID++
	return true, nil
}

func (r *memoryCheckInRepository) ListByMonth(_ context.Context, userID int64, year int, month int) ([]do.CheckIn, error) {
	prefix := fmt.Sprintf("%04d-%02d-", year, month)
	var result []do.CheckIn
	for _, record := range r.records {
		if record.UserID != userID {
			continue
		}
		if strings.HasPrefix(record.CheckInDate, prefix) {
			result = append(result, record)
		}
	}
	return result, nil
}

var _ ports.CheckInRepository = (*memoryCheckInRepository)(nil)

func Test_CheckInService_CheckInToday_awards_points_on_first_check_in(t *testing.T) {
	// Given
	userRepo := newMemoryUserRepository()
	userRepo.byID[1] = do.User{ID: 1, Username: "alice"}
	checkInRepo := newMemoryCheckInRepository()
	svc := service.NewCheckInService(checkInRepo, userRepo)

	// When
	result, err := svc.CheckInToday(context.Background(), 1)

	// Then
	if err != nil {
		t.Fatalf("check in today: %v", err)
	}
	if !result.CheckedIn {
		t.Fatalf("CheckedIn = false, want true on first check-in")
	}
	if result.PointsAwarded != 10 {
		t.Fatalf("PointsAwarded = %d, want 10", result.PointsAwarded)
	}
}

func Test_CheckInService_CheckInToday_is_idempotent_on_repeat(t *testing.T) {
	// Given
	userRepo := newMemoryUserRepository()
	userRepo.byID[1] = do.User{ID: 1, Username: "alice"}
	checkInRepo := newMemoryCheckInRepository()
	svc := service.NewCheckInService(checkInRepo, userRepo)
	_, err := svc.CheckInToday(context.Background(), 1)
	if err != nil {
		t.Fatalf("first check in today: %v", err)
	}

	// When
	result, err := svc.CheckInToday(context.Background(), 1)

	// Then
	if err != nil {
		t.Fatalf("repeat check in today: %v", err)
	}
	if result.CheckedIn {
		t.Fatalf("CheckedIn = true, want false on repeat check-in")
	}
	if result.PointsAwarded != 0 {
		t.Fatalf("PointsAwarded = %d, want 0 on repeat check-in", result.PointsAwarded)
	}
}

func Test_CheckInService_ListMonthly_returns_dates_and_total_points(t *testing.T) {
	// Given
	userRepo := newMemoryUserRepository()
	userRepo.byID[1] = do.User{ID: 1, Username: "alice", Points: 30}
	checkInRepo := newMemoryCheckInRepository()
	svc := service.NewCheckInService(checkInRepo, userRepo)

	// When
	result, err := svc.ListMonthly(context.Background(), 1, 2026, 7)

	// Then
	if err != nil {
		t.Fatalf("list monthly: %v", err)
	}
	if result.TotalPoints != 30 {
		t.Fatalf("TotalPoints = %d, want 30", result.TotalPoints)
	}
	if len(result.Dates) != 0 {
		t.Fatalf("Dates len = %d, want 0 for empty month", len(result.Dates))
	}
}

func Test_CheckInService_ListMonthly_returns_error_when_user_not_found(t *testing.T) {
	// Given
	userRepo := newMemoryUserRepository()
	checkInRepo := newMemoryCheckInRepository()
	svc := service.NewCheckInService(checkInRepo, userRepo)

	// When
	_, err := svc.ListMonthly(context.Background(), 999, 2026, 7)

	// Then
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("error = %v, want ErrUserNotFound", err)
	}
}
