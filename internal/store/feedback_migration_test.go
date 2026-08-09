package store

import (
	"testing"
	"time"
)

func TestAutoMigrateCreatesFeedbackTableAndDefaultsPending(t *testing.T) {
	db := openTestDB(t)

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if !db.Migrator().HasTable("feedbacks") {
		t.Fatalf("expected feedbacks table to exist")
	}

	feedback := Feedback{
		ID:        "fb_migration",
		ProductID: "prd_migration",
		Category:  "bug",
		Content:   "Migration check",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(&feedback).Error; err != nil {
		t.Fatalf("create feedback: %v", err)
	}
	if feedback.Status != "pending" {
		t.Fatalf("expected default status pending, got %q", feedback.Status)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}
}
