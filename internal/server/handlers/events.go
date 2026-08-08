package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/store"
)

type postEventReq struct {
	Type      string `json:"type"`
	VariantID string `json:"variant_id"`
}

// PostEvent increments the variant visit counter for client-side visit beacons.
func PostEvent(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in postEventReq
		if err := c.ShouldBindJSON(&in); err != nil || in.Type != "visit" || in.VariantID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": gin.H{"code": "BAD_REQUEST"}})
			return
		}
		_ = db.Model(&store.Variant{}).Where("id = ?", in.VariantID).
			UpdateColumn("visit_count", gorm.Expr("visit_count + 1")).Error
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	}
}
