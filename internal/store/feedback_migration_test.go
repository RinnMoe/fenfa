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
	for _, index := range []string{
		"idx_feedback_created",
		"idx_feedback_product_created",
		"idx_feedback_category_created",
		"idx_feedback_status_created",
	} {
		if !db.Migrator().HasIndex(&Feedback{}, index) {
			t.Fatalf("expected feedback index %q to exist", index)
		}
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

func TestAutoMigrateInitializesDefaultEmailSettings(t *testing.T) {
	db := openTestDB(t)

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	var settings SystemSettings
	if err := db.Where("id = ?", "default").First(&settings).Error; err != nil {
		t.Fatalf("load default settings: %v", err)
	}
	if settings.EmailEnabled {
		t.Fatalf("expected email notifications to be disabled by default")
	}
	if settings.SMTPPort != 587 || settings.SMTPEncryption != "starttls" {
		t.Fatalf("unexpected default SMTP settings: port=%d encryption=%q", settings.SMTPPort, settings.SMTPEncryption)
	}
}
