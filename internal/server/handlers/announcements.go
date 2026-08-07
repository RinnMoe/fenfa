package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

type adminAnnouncementInput struct {
	Title   string     `json:"title"`
	Content string     `json:"content"`
	Enabled *bool      `json:"enabled"`
	StartAt *time.Time `json:"start_at"`
	EndAt   *time.Time `json:"end_at"`
}

func AdminListAnnouncements(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		productID := c.Param("productID")
		var product store.Product
		if err := db.Where("id = ?", productID).First(&product).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "product not found"}})
			return
		}

		var announcements []store.Announcement
		if err := db.Where("product_id = ?", product.ID).Order("created_at desc").Find(&announcements).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": err.Error()}})
			return
		}

		items := make([]gin.H, 0, len(announcements))
		for _, announcement := range announcements {
			items = append(items, announcementView(announcement))
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"items": items, "total": len(items)}})
	}
}

func AdminGetAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		announcementID := c.Param("announcementID")
		var announcement store.Announcement
		if err := db.Where("id = ?", announcementID).First(&announcement).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "announcement not found"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"announcement": announcementView(announcement)}})
	}
}

func AdminCreateAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		productID := c.Param("productID")
		var input adminAnnouncementInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "invalid announcement payload"}})
			return
		}
		if message := validateAnnouncementInput(input); message != "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": message}})
			return
		}

		var product store.Product
		if err := db.Where("id = ?", productID).First(&product).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "product not found"}})
			return
		}

		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		now := time.Now()
		announcement := store.Announcement{
			ID:        "ann_" + randHexN(8),
			ProductID: product.ID,
			Title:     strings.TrimSpace(input.Title),
			Content:   input.Content,
			Enabled:   enabled,
			StartAt:   input.StartAt,
			EndAt:     input.EndAt,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := db.Select("*").Create(&announcement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": err.Error()}})
			return
		}
		if announcement.Enabled != enabled {
			if err := db.Model(&announcement).UpdateColumn("enabled", enabled).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": err.Error()}})
				return
			}
			announcement.Enabled = enabled
		}
		c.JSON(http.StatusCreated, gin.H{"ok": true, "data": gin.H{"announcement": announcementView(announcement)}})
	}
}

func AdminUpdateAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		announcementID := c.Param("announcementID")
		var input adminAnnouncementInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "invalid announcement payload"}})
			return
		}
		if message := validateAnnouncementInput(input); message != "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": message}})
			return
		}

		var announcement store.Announcement
		if err := db.Where("id = ?", announcementID).First(&announcement).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "announcement not found"}})
			return
		}

		enabled := announcement.Enabled
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		updates := map[string]any{
			"title":      strings.TrimSpace(input.Title),
			"content":    input.Content,
			"enabled":    enabled,
			"start_at":   input.StartAt,
			"end_at":     input.EndAt,
			"updated_at": time.Now(),
		}
		if err := db.Model(&announcement).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": err.Error()}})
			return
		}

		if err := db.Where("id = ?", announcementID).First(&announcement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"announcement": announcementView(announcement)}})
	}
}

func AdminDeleteAnnouncement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		announcementID := c.Param("announcementID")
		var announcement store.Announcement
		if err := db.Where("id = ?", announcementID).First(&announcement).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "announcement not found"}})
			return
		}
		if err := db.Delete(&announcement).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func validateAnnouncementInput(input adminAnnouncementInput) string {
	if strings.TrimSpace(input.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(input.Content) == "" {
		return "content is required"
	}
	if input.StartAt != nil && input.EndAt != nil && input.StartAt.After(*input.EndAt) {
		return "start_at must not be after end_at"
	}
	return ""
}

func announcementView(announcement store.Announcement) gin.H {
	return gin.H{
		"id":         announcement.ID,
		"product_id": announcement.ProductID,
		"title":      announcement.Title,
		"content":    announcement.Content,
		"enabled":    announcement.Enabled,
		"start_at":   announcement.StartAt,
		"end_at":     announcement.EndAt,
		"created_at": announcement.CreatedAt,
		"updated_at": announcement.UpdatedAt,
	}
}
