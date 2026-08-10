package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

const (
	feedbackStatusPending      = "pending"
	feedbackStatusProcessed    = "processed"
	feedbackCategoryBug        = "bug"
	feedbackCategorySuggest    = "suggestion"
	feedbackCategoryOther      = "other"
	feedbackRateWindow         = 10 * time.Minute
	feedbackRateLimit          = 5
	feedbackRequestRateLimit   = 30
	feedbackRateLimiterMaxKeys = 10000
	feedbackMaxBodyBytes       = 16 * 1024
	feedbackMaxContentRunes    = 4000
	feedbackMaxContactRunes    = 255
)

type publicFeedbackInput struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	ReleaseID string `json:"release_id"`
	Category  string `json:"category"`
	Content   string `json:"content"`
	Contact   string `json:"contact"`
	Website   string `json:"website"`
}

type adminFeedbackUpdateInput struct {
	Status string `json:"status"`
}

type FeedbackNotifier interface {
	NotifyFeedback(store.Feedback)
}

type feedbackRateLimiter struct {
	mu          sync.Mutex
	entries     map[string][]time.Time
	lastCleanup time.Time
	limit       int
	maxKeys     int
}

func newFeedbackRateLimiter(limit int) *feedbackRateLimiter {
	return &feedbackRateLimiter{
		entries:     make(map[string][]time.Time),
		lastCleanup: time.Now(),
		limit:       limit,
		maxKeys:     feedbackRateLimiterMaxKeys,
	}
}

func (limiter *feedbackRateLimiter) allow(ip string, now time.Time) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	cutoff := now.Add(-feedbackRateWindow)
	if now.Sub(limiter.lastCleanup) >= time.Minute || len(limiter.entries) >= limiter.maxKeys {
		for key, timestamps := range limiter.entries {
			hasRecent := false
			for _, timestamp := range timestamps {
				if timestamp.After(cutoff) {
					hasRecent = true
					break
				}
			}
			if !hasRecent {
				delete(limiter.entries, key)
			}
		}
		limiter.lastCleanup = now
	}
	previous := limiter.entries[ip]
	wasTracked := len(previous) > 0
	kept := previous[:0]
	for _, timestamp := range previous {
		if timestamp.After(cutoff) {
			kept = append(kept, timestamp)
		}
	}
	if len(kept) == 0 {
		delete(limiter.entries, ip)
		wasTracked = false
	} else {
		limiter.entries[ip] = kept
	}

	if len(kept) >= limiter.limit {
		oldest := kept[0]
		for _, timestamp := range kept[1:] {
			if timestamp.Before(oldest) {
				oldest = timestamp
			}
		}
		retryAfter := oldest.Add(feedbackRateWindow).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter
	}
	if !wasTracked && len(limiter.entries) >= limiter.maxKeys {
		return false, feedbackRateWindow
	}

	limiter.entries[ip] = append(kept, now)
	return true, 0
}

// PublicCreateFeedback accepts anonymous feedback from a published product page.
func PublicCreateFeedback(db *gorm.DB, notifiers ...FeedbackNotifier) gin.HandlerFunc {
	requestLimiter := newFeedbackRateLimiter(feedbackRequestRateLimit)
	submissionLimiter := newFeedbackRateLimiter(feedbackRateLimit)
	var notifier FeedbackNotifier
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}

	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, feedbackMaxBodyBytes)
		if allowed, retryAfter := requestLimiter.allow(c.ClientIP(), time.Now()); !allowed {
			writeFeedbackRateLimitResponse(c, retryAfter)
			return
		}

		var input publicFeedbackInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "invalid feedback payload"}})
			return
		}

		// The honeypot is intentionally silent so automated submitters do not get
		// a useful signal that their request was rejected.
		if strings.TrimSpace(input.Website) != "" {
			c.JSON(http.StatusCreated, gin.H{"ok": true, "data": gin.H{"accepted": true}})
			return
		}

		productID := strings.TrimSpace(input.ProductID)
		variantID := strings.TrimSpace(input.VariantID)
		releaseID := strings.TrimSpace(input.ReleaseID)
		category := strings.TrimSpace(strings.ToLower(input.Category))
		content := strings.TrimSpace(input.Content)
		contact := strings.TrimSpace(input.Contact)

		if productID == "" || !validFeedbackCategory(category) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "product_id and a valid category are required"}})
			return
		}
		if content == "" || utf8.RuneCountInString(content) > feedbackMaxContentRunes {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "content is required and must be at most 4000 characters"}})
			return
		}
		if utf8.RuneCountInString(contact) > feedbackMaxContactRunes {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "contact must be at most 255 characters"}})
			return
		}

		var product store.Product
		if err := db.Where("id = ? AND published = ?", productID, true).First(&product).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "product not found"}})
			return
		}

		var variant store.Variant
		if variantID != "" {
			if err := db.Where("id = ? AND product_id = ? AND published = ?", variantID, product.ID, true).First(&variant).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "variant not found"}})
				return
			}
		}

		var release store.Release
		if releaseID != "" {
			if err := db.Where("id = ?", releaseID).First(&release).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "release not found"}})
				return
			}
			if variantID != "" && release.VariantID != variant.ID {
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "release does not belong to variant"}})
				return
			}
			if variantID == "" {
				if err := db.Where("id = ? AND product_id = ? AND published = ?", release.VariantID, product.ID, true).First(&variant).Error; err != nil {
					c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "variant not found"}})
					return
				}
				variantID = variant.ID
			}
		}
		if allowed, retryAfter := submissionLimiter.allow(c.ClientIP(), time.Now()); !allowed {
			writeFeedbackRateLimitResponse(c, retryAfter)
			return
		}

		now := time.Now().UTC()
		feedback := store.Feedback{
			ID:              "fb_" + randHexN(8),
			ProductID:       product.ID,
			VariantID:       variantID,
			ReleaseID:       releaseID,
			ProductName:     product.Name,
			VariantPlatform: variant.Platform,
			ReleaseVersion:  release.Version,
			Category:        category,
			Content:         content,
			Contact:         contact,
			Status:          feedbackStatusPending,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := db.Create(&feedback).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": "failed to save feedback"}})
			return
		}
		if notifier != nil {
			notifier.NotifyFeedback(feedback)
		}

		c.JSON(http.StatusCreated, gin.H{"ok": true, "data": gin.H{
			"feedback": gin.H{
				"id":         feedback.ID,
				"status":     feedback.Status,
				"created_at": feedback.CreatedAt,
			},
		}})
	}
}

func writeFeedbackRateLimitResponse(c *gin.Context, retryAfter time.Duration) {
	seconds := int((retryAfter + time.Second - 1) / time.Second)
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.JSON(http.StatusTooManyRequests, gin.H{"ok": false, "error": gin.H{"code": "RATE_LIMITED", "message": "too many feedback submissions"}})
}

// AdminListFeedback returns paginated feedback for the authenticated admin.
func AdminListFeedback(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, pageSize, ok := feedbackPagination(c)
		if !ok {
			return
		}

		status := strings.TrimSpace(c.Query("status"))
		category := strings.TrimSpace(strings.ToLower(c.Query("category")))
		productID := strings.TrimSpace(c.Query("product_id"))
		if status != "" && !validFeedbackStatus(status) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "invalid status"}})
			return
		}
		if category != "" && !validFeedbackCategory(category) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "invalid category"}})
			return
		}

		query := db.Model(&store.Feedback{})
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if category != "" {
			query = query.Where("category = ?", category)
		}
		if productID != "" {
			query = query.Where("product_id = ?", productID)
		}

		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": "failed to count feedback"}})
			return
		}

		var feedbacks []store.Feedback
		if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&feedbacks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": "failed to load feedback"}})
			return
		}

		items := make([]gin.H, 0, len(feedbacks))
		for _, feedback := range feedbacks {
			items = append(items, feedbackAdminView(feedback))
		}

		totalPages := 0
		if total > 0 {
			totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{
			"items":       items,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		}})
	}
}

// AdminUpdateFeedback updates the processing state of one feedback item.
func AdminUpdateFeedback(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input adminFeedbackUpdateInput
		if err := c.ShouldBindJSON(&input); err != nil || !validFeedbackStatus(strings.TrimSpace(input.Status)) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "status must be pending or processed"}})
			return
		}

		feedbackID := c.Param("feedbackID")
		var feedback store.Feedback
		if err := db.Where("id = ?", feedbackID).First(&feedback).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "feedback not found"}})
			return
		}

		if err := db.Model(&feedback).Updates(map[string]any{
			"status":     strings.TrimSpace(input.Status),
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": "failed to update feedback"}})
			return
		}
		if err := db.Where("id = ?", feedbackID).First(&feedback).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": "failed to reload feedback"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"feedback": feedbackAdminView(feedback)}})
	}
}

// AdminDeleteFeedback permanently deletes one feedback item.
func AdminDeleteFeedback(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		feedbackID := c.Param("feedbackID")
		var feedback store.Feedback
		if err := db.Where("id = ?", feedbackID).First(&feedback).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "feedback not found"}})
			return
		}
		if err := db.Delete(&feedback).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": "failed to delete feedback"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func feedbackPagination(c *gin.Context) (int, int, bool) {
	page := 1
	pageSize := 20
	var err error
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "page must be a positive integer"}})
			return 0, 0, false
		}
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil || pageSize < 1 || pageSize > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "page_size must be between 1 and 100"}})
			return 0, 0, false
		}
	}
	return page, pageSize, true
}

func validFeedbackCategory(category string) bool {
	switch category {
	case feedbackCategoryBug, feedbackCategorySuggest, feedbackCategoryOther:
		return true
	default:
		return false
	}
}

func validFeedbackStatus(status string) bool {
	return status == feedbackStatusPending || status == feedbackStatusProcessed
}

func feedbackAdminView(feedback store.Feedback) gin.H {
	return gin.H{
		"id":               feedback.ID,
		"product_id":       feedback.ProductID,
		"variant_id":       feedback.VariantID,
		"release_id":       feedback.ReleaseID,
		"product_name":     feedback.ProductName,
		"variant_platform": feedback.VariantPlatform,
		"release_version":  feedback.ReleaseVersion,
		"category":         feedback.Category,
		"content":          feedback.Content,
		"contact":          feedback.Contact,
		"status":           feedback.Status,
		"created_at":       feedback.CreatedAt,
		"updated_at":       feedback.UpdatedAt,
	}
}
