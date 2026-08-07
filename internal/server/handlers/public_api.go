package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/store"
)

func PublicLatestRelease(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("slug"))
		var product store.Product
		if err := db.Where("slug = ? AND published = ?", slug, true).First(&product).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "product not found"}})
			return
		}

		variantID := strings.TrimSpace(c.Query("variant"))
		var variant store.Variant
		if variantID == "" {
			var variantCount int64
			db.Model(&store.Variant{}).Where("product_id = ? AND published = ?", product.ID, true).Count(&variantCount)
			if variantCount != 1 {
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST", "message": "variant is required when a product has multiple variants"}})
				return
			}
			if err := db.Where("product_id = ? AND published = ?", product.ID, true).Order("sort_order asc, platform asc").First(&variant).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "variant not found"}})
				return
			}
		} else if err := db.Where("id = ? AND product_id = ? AND published = ?", variantID, product.ID, true).First(&variant).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "variant not found"}})
			return
		}

		var release store.Release
		if err := db.Where("variant_id = ?", variant.ID).Order("created_at desc").First(&release).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "release not found"}})
			return
		}

		domains := resolvePublicDomains(c, db, cfg)
		actions := releaseActions(productPathKey(product), variant.Platform, domains, release.ID)
		downloadURL, _ := actions["download"].(string)
		releasePage, _ := actions["release_page"].(string)

		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{
			"product": gin.H{
				"id":        product.ID,
				"slug":      product.Slug,
				"name":      product.Name,
				"published": product.Published,
			},
			"variant": gin.H{
				"id":             variant.ID,
				"product_id":     variant.ProductID,
				"platform":       variant.Platform,
				"identifier":     variant.Identifier,
				"display_name":   variant.DisplayName,
				"arch":           variant.Arch,
				"installer_type": variant.InstallerType,
				"min_os":         variant.MinOS,
				"published":      variant.Published,
			},
			"release": gin.H{
				"id":           release.ID,
				"version":      release.Version,
				"build":        release.Build,
				"changelog":    release.Changelog,
				"force_update": release.ForceUpdate,
				"created_at":   release.CreatedAt,
				"min_os":       release.MinOS,
				"channel":      release.Channel,
				"download_url": downloadURL,
				"release_page": releasePage,
				"actions":      actions,
			},
		}})
	}
}

func PublicAnnouncements(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("slug"))
		var product store.Product
		if err := db.Where("slug = ? AND published = ?", slug, true).First(&product).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": gin.H{"code": "NOT_FOUND", "message": "product not found"}})
			return
		}

		now := time.Now().UTC()
		var announcements []store.Announcement
		if err := db.Where(
			"product_id = ? AND enabled = ? AND (start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)",
			product.ID, true, now, now,
		).Order("created_at desc").Find(&announcements).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": gin.H{"code": "DB", "message": err.Error()}})
			return
		}

		items := make([]gin.H, 0, len(announcements))
		for _, announcement := range announcements {
			items = append(items, gin.H{
				"id":       announcement.ID,
				"title":    announcement.Title,
				"content":  announcement.Content,
				"start_at": announcement.StartAt,
				"end_at":   announcement.EndAt,
			})
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"items": items}})
	}
}
