package email

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

type recordingFeedbackSender struct {
	calls   chan Message
	err     error
	configs chan SMTPConfig
}

func (sender *recordingFeedbackSender) Send(_ context.Context, config SMTPConfig, message Message) error {
	sender.configs <- config
	sender.calls <- message
	return sender.err
}

func openFeedbackNotifierTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/feedback.db"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := store.AutoMigrate(db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get test database handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func configureFeedbackEmail(t *testing.T, db *gorm.DB, enabled bool) {
	t.Helper()
	updates := map[string]any{
		"email_enabled":   enabled,
		"smtp_host":       "smtp.example.com",
		"smtp_port":       587,
		"smtp_encryption": EncryptionStartTLS,
		"smtp_from":       "noreply@example.com",
		"smtp_to":         "admin@example.com",
	}
	if err := db.Model(&store.SystemSettings{}).Where("id = ?", "default").Updates(updates).Error; err != nil {
		t.Fatalf("configure feedback email: %v", err)
	}
}

func TestFeedbackNotifierSkipsDisabledOrInvalidSettings(t *testing.T) {
	for _, test := range []struct {
		name    string
		enabled bool
		valid   bool
	}{
		{name: "disabled", enabled: false, valid: true},
		{name: "invalid enabled configuration", enabled: true, valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openFeedbackNotifierTestDB(t)
			configureFeedbackEmail(t, db, test.enabled)
			if !test.valid {
				if err := db.Model(&store.SystemSettings{}).Where("id = ?", "default").Update("smtp_host", "").Error; err != nil {
					t.Fatalf("invalidate SMTP settings: %v", err)
				}
			}

			sender := &recordingFeedbackSender{calls: make(chan Message, 1), configs: make(chan SMTPConfig, 1)}
			notifier := NewFeedbackNotifier(db, sender)
			notifier.NotifyFeedback(store.Feedback{ID: "fb_skip"})
			defer notifier.Close()

			select {
			case message := <-sender.calls:
				t.Fatalf("unexpected notification message: %+v", message)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

func TestFeedbackNotifierSendsFeedbackAndRetriesFailures(t *testing.T) {
	db := openFeedbackNotifierTestDB(t)
	configureFeedbackEmail(t, db, true)

	sender := &recordingFeedbackSender{
		calls:   make(chan Message, feedbackSendTries),
		configs: make(chan SMTPConfig, feedbackSendTries),
		err:     errors.New("fake SMTP failure"),
	}
	notifier := newFeedbackNotifier(db, sender, func(time.Duration) {})
	defer notifier.Close()
	notifier.NotifyFeedback(store.Feedback{
		ID:              "fb_retry",
		ProductName:     "Example App",
		VariantPlatform: "ios",
		ReleaseVersion:  "1.2.3",
		Category:        "bug",
		Contact:         "user@example.com",
		Content:         "The app does not start",
		CreatedAt:       time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC),
	})

	for attempt := 0; attempt < feedbackSendTries; attempt++ {
		select {
		case message := <-sender.calls:
			if message.Subject != "[Fenfa] New feedback received" {
				t.Fatalf("unexpected feedback subject: %q", message.Subject)
			}
			for _, expected := range []string{"Feedback ID: fb_retry", "Product: Example App", "The app does not start"} {
				if !strings.Contains(message.Body, expected) {
					t.Fatalf("feedback message missing %q: %q", expected, message.Body)
				}
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for retry %d", attempt+1)
		}
	}
}
