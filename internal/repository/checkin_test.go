package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

func Test_CheckInRepository_CheckInToday_awards_points_on_first_checkin(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	userRepo := repository.NewUserRepository(database.Read, database.Write)
	checkInRepo := repository.NewCheckInRepository(database.Read, database.Write)
	user, err := userRepo.Create(context.Background(), do.User{
		Username:     "checkin-user",
		PasswordHash: "hash",
		Role:         do.UserRoleUser,
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// When: first check-in
	checkedIn, err := checkInRepo.CheckInToday(context.Background(), user.ID, "2026-07-08", 10)

	// Then
	if err != nil {
		t.Fatalf("check in today: %v", err)
	}
	if !checkedIn {
		t.Fatalf("checkedIn = false, want true on first check-in")
	}
	updated, _ := userRepo.FindByID(context.Background(), user.ID)
	if updated.Points != 10 {
		t.Fatalf("points = %d, want 10", updated.Points)
	}
}

func Test_CheckInRepository_CheckInToday_idempotent_on_same_date(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	userRepo := repository.NewUserRepository(database.Read, database.Write)
	checkInRepo := repository.NewCheckInRepository(database.Read, database.Write)
	user, _ := userRepo.Create(context.Background(), do.User{
		Username: "checkin-idem", PasswordHash: "hash", Role: do.UserRoleUser, CreatedAt: time.Now().UTC(),
	})
	checkInRepo.CheckInToday(context.Background(), user.ID, "2026-07-08", 10)

	// When: second check-in on same date
	checkedIn, err := checkInRepo.CheckInToday(context.Background(), user.ID, "2026-07-08", 10)

	// Then
	if err != nil {
		t.Fatalf("second check in: %v", err)
	}
	if checkedIn {
		t.Fatalf("checkedIn = true, want false on duplicate date")
	}
	updated, _ := userRepo.FindByID(context.Background(), user.ID)
	if updated.Points != 10 {
		t.Fatalf("points = %d, want 10 (not doubled)", updated.Points)
	}
}

func Test_CheckInRepository_CheckInToday_allows_different_dates(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	userRepo := repository.NewUserRepository(database.Read, database.Write)
	checkInRepo := repository.NewCheckInRepository(database.Read, database.Write)
	user, _ := userRepo.Create(context.Background(), do.User{
		Username: "checkin-multi", PasswordHash: "hash", Role: do.UserRoleUser, CreatedAt: time.Now().UTC(),
	})

	// When: check-in on two different dates
	checkInRepo.CheckInToday(context.Background(), user.ID, "2026-07-07", 10)
	checkInRepo.CheckInToday(context.Background(), user.ID, "2026-07-08", 10)

	// Then
	updated, _ := userRepo.FindByID(context.Background(), user.ID)
	if updated.Points != 20 {
		t.Fatalf("points = %d, want 20", updated.Points)
	}
}

func Test_CheckInRepository_ListByMonth_returns_dates_ascending(t *testing.T) {
	// Given
	database := openTestDatabase(t)
	userRepo := repository.NewUserRepository(database.Read, database.Write)
	checkInRepo := repository.NewCheckInRepository(database.Read, database.Write)
	user, _ := userRepo.Create(context.Background(), do.User{
		Username: "checkin-list", PasswordHash: "hash", Role: do.UserRoleUser, CreatedAt: time.Now().UTC(),
	})
	checkInRepo.CheckInToday(context.Background(), user.ID, "2026-07-08", 10)
	checkInRepo.CheckInToday(context.Background(), user.ID, "2026-07-01", 10)
	checkInRepo.CheckInToday(context.Background(), user.ID, "2026-06-30", 10)

	// When
	records, err := checkInRepo.ListByMonth(context.Background(), user.ID, 2026, 7)

	// Then
	if err != nil {
		t.Fatalf("list by month: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records count = %d, want 2", len(records))
	}
	if records[0].CheckInDate != "2026-07-01" || records[1].CheckInDate != "2026-07-08" {
		t.Fatalf("dates = %s, %s, want ascending order", records[0].CheckInDate, records[1].CheckInDate)
	}
}
