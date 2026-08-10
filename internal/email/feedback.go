package email

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

const (
	feedbackQueueSize   = 64
	feedbackSendTries   = 3
	feedbackSendTimeout = 15 * time.Second
)

// FeedbackNotifier asynchronously sends email notifications for persisted
// feedback. It intentionally uses an in-process queue; pending items are not
// persisted across process restarts.
type FeedbackNotifier struct {
	db     *gorm.DB
	sender Sender
	queue  chan store.Feedback
	stop   chan struct{}
	sleep  func(time.Duration)
	once   sync.Once
	wg     sync.WaitGroup
}

func NewFeedbackNotifier(db *gorm.DB, sender Sender) *FeedbackNotifier {
	return newFeedbackNotifier(db, sender, time.Sleep)
}

func newFeedbackNotifier(db *gorm.DB, sender Sender, sleep func(time.Duration)) *FeedbackNotifier {
	if sender == nil {
		sender = SMTPMailer{}
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	notifier := &FeedbackNotifier{
		db:     db,
		sender: sender,
		queue:  make(chan store.Feedback, feedbackQueueSize),
		stop:   make(chan struct{}),
		sleep:  sleep,
	}
	notifier.wg.Add(1)
	go notifier.run()
	return notifier
}

func (notifier *FeedbackNotifier) NotifyFeedback(feedback store.Feedback) {
	select {
	case <-notifier.stop:
		return
	case notifier.queue <- feedback:
	default:
		log.Printf("[email] feedback notification queue is full; feedback_id=%s", feedback.ID)
	}
}

func (notifier *FeedbackNotifier) Close() {
	notifier.once.Do(func() {
		close(notifier.stop)
	})
	notifier.wg.Wait()
}

func (notifier *FeedbackNotifier) run() {
	defer notifier.wg.Done()
	for {
		select {
		case <-notifier.stop:
			return
		case feedback := <-notifier.queue:
			notifier.send(feedback)
		}
	}
}

func (notifier *FeedbackNotifier) send(feedback store.Feedback) {
	var settings store.SystemSettings
	if err := notifier.db.Where("id = ?", "default").First(&settings).Error; err != nil {
		log.Printf("[email] could not load email settings; feedback_id=%s", feedback.ID)
		return
	}
	if !settings.EmailEnabled {
		return
	}

	config := SMTPConfigFromSettings(settings)
	if err := ValidateSMTPConfig(config); err != nil {
		log.Printf("[email] feedback notification skipped because SMTP settings are invalid; feedback_id=%s", feedback.ID)
		return
	}

	message := Message{
		Subject: "[Fenfa] New feedback received",
		Body:    formatFeedbackBody(feedback),
	}
	for attempt := 1; attempt <= feedbackSendTries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), feedbackSendTimeout)
		err := notifier.sender.Send(ctx, config, message)
		cancel()
		if err == nil {
			return
		}
		if attempt < feedbackSendTries {
			notifier.sleep(time.Duration(attempt) * time.Second)
		}
	}
	log.Printf("[email] failed to send feedback notification; feedback_id=%s", feedback.ID)
}

func formatFeedbackBody(feedback store.Feedback) string {
	var body strings.Builder
	body.WriteString("A new feedback was submitted to Fenfa.\n\n")
	fmt.Fprintf(&body, "Feedback ID: %s\n", feedback.ID)
	fmt.Fprintf(&body, "Product: %s\n", feedback.ProductName)
	fmt.Fprintf(&body, "Platform: %s\n", feedback.VariantPlatform)
	fmt.Fprintf(&body, "Version: %s\n", feedback.ReleaseVersion)
	fmt.Fprintf(&body, "Category: %s\n", feedback.Category)
	fmt.Fprintf(&body, "Contact: %s\n", feedback.Contact)
	fmt.Fprintf(&body, "Received at: %s\n\n", feedback.CreatedAt.UTC().Format(time.RFC3339))
	body.WriteString("Content:\n")
	body.WriteString(feedback.Content)
	body.WriteByte('\n')
	return body.String()
}
