package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

const (
	dbPath         = "data/acgwarehouse.db"
	testUserPrefix = "loadtest_"
	testPassword   = "Test123456"
	numTestUsers   = 3
	numRatings     = 100
	numCollections = 3
)

func main() {
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	db, err := openDB()
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	// Check existing images
	var imageCount int64
	if err := db.Model(&po.Image{}).Where("status = ?", "active").Count(&imageCount).Error; err != nil {
		log.Fatalf("count images: %v", err)
	}
	fmt.Printf("Found %d active images\n", imageCount)

	if imageCount < 10 {
		log.Fatalf("not enough images for testing (need at least 10)")
	}

	// Get image IDs
	var imageIDs []int64
	if err := db.Model(&po.Image{}).Where("status = ?", "active").Limit(200).Pluck("id", &imageIDs).Error; err != nil {
		log.Fatalf("get image ids: %v", err)
	}

	// Create test users
	users := createTestUsers(ctx, db, numTestUsers)
	fmt.Printf("Created %d test users\n", len(users))

	// Create ratings
	for _, user := range users {
		for i := 0; i < numRatings; i++ {
			imageID := imageIDs[rand.Intn(len(imageIDs))]
			score := rand.Intn(101) // 0-100
			if err := createRating(ctx, db, user.ID, imageID, score); err != nil {
				log.Printf("create rating failed: user=%d, image=%d, err=%v", user.ID, imageID, err)
			}
		}
	}
	fmt.Printf("Created %d ratings per user\n", numRatings)

	// Create collections and items
	for _, user := range users {
		for i := 0; i < numCollections; i++ {
			name := fmt.Sprintf("%s收藏夹%d", user.Username, i+1)
			collection, err := createCollection(ctx, db, user.ID, name)
			if err != nil {
				log.Printf("create collection failed: user=%d, err=%v", user.ID, err)
				continue
			}
			// Add random images to collection
			for j := 0; j < 10; j++ {
				imageID := imageIDs[rand.Intn(len(imageIDs))]
				if err := addCollectionItem(ctx, db, collection.ID, imageID); err != nil {
					log.Printf("add collection item failed: collection=%d, image=%d, err=%v", collection.ID, imageID, err)
				}
			}
		}
	}
	fmt.Printf("Created %d collections per user with 10 items each\n", numCollections)

	fmt.Println("Test data setup completed!")
}

func openDB() (*gorm.DB, error) {
	return gorm.Open(&sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func createTestUsers(ctx context.Context, db *gorm.DB, count int) []po.User {
	users := make([]po.User, 0, count)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	for i := 1; i <= count; i++ {
		username := fmt.Sprintf("%suser%d", testUserPrefix, i)
		role := "user"
		if i == count {
			username = fmt.Sprintf("%sadmin", testUserPrefix)
			role = "admin"
		}

		user := po.User{
			Username:     username,
			PasswordHash: string(hashedPassword),
			Role:         role,
			CreatedAt:    time.Now().UTC(),
		}

		// Upsert: if exists, skip
		result := db.WithContext(ctx).Where("username = ?", username).FirstOrCreate(&user)
		if result.Error != nil {
			log.Printf("create user %s: %v", username, result.Error)
			continue
		}
		users = append(users, user)
	}
	return users
}

func createRating(ctx context.Context, db *gorm.DB, userID, imageID int64, score int) error {
	now := time.Now().UTC()
	return db.WithContext(ctx).Exec(`
		INSERT INTO rating (user_id, image_id, score, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, image_id) DO UPDATE SET score = ?, updated_at = ?
	`, userID, imageID, score, now, score, now).Error
}

func createCollection(ctx context.Context, db *gorm.DB, userID int64, name string) (po.Collection, error) {
	collection := po.Collection{
		UserID:     userID,
		Name:       name,
		Visibility: "private",
		CreatedAt:  time.Now().UTC(),
	}
	err := db.WithContext(ctx).Create(&collection).Error
	return collection, err
}

func addCollectionItem(ctx context.Context, db *gorm.DB, collectionID, imageID int64) error {
	now := time.Now().UTC()
	return db.WithContext(ctx).Exec(`
		INSERT OR IGNORE INTO collection_item (collection_id, image_id, created_at)
		VALUES (?, ?, ?)
	`, collectionID, imageID, now).Error
}
