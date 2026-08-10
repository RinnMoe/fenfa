package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/openprx/fenfa/internal/config"
	"github.com/openprx/fenfa/internal/server/handlers"
	"github.com/openprx/fenfa/internal/server/middleware"
)

func RegisterRoutes(g *gin.Engine, db *gorm.DB, cfg *config.Config, notifiers ...handlers.FeedbackNotifier) error {
	if err := g.SetTrustedProxies(cfg.GetTrustedProxies()); err != nil {
		return err
	}
	adminPath := cfg.GetAdminPath()
	var feedbackNotifier handlers.FeedbackNotifier
	if len(notifiers) > 0 {
		feedbackNotifier = notifiers[0]
	}

	// Static served in main, still ok to keep here if needed
	g.GET("/healthz", handlers.Healthz(db))

	// Public HTML + JSON
	g.GET("/apps/:appID", func(c *gin.Context) {
		handlers.LegacyAppRedirect(db)(c)
	})
	g.GET("/products/:productID", func(c *gin.Context) {
		if cfg.Server.DevProxyFront != "" {
			target := cfg.Server.DevProxyFront + c.Request.URL.Path
			if q := c.Request.URL.RawQuery; q != "" {
				target += "?" + q
			}
			c.Redirect(http.StatusTemporaryRedirect, target)
			return
		}
		handlers.ProductDetail(db)(c)
	})
	g.GET("/api/apps/:appID", handlers.AppDetailJSON(db, cfg))
	g.GET("/api/products/:productID", handlers.ProductDetailJSON(db, cfg))
	g.GET("/api/products/slug/:slug", handlers.ProductDetailJSON(db, cfg))
	g.GET("/api/v1/products/:slug/latest", handlers.PublicLatestRelease(db, cfg))
	g.GET("/api/v1/products/:slug/announcements", handlers.PublicAnnouncements(db))
	g.POST("/api/v1/feedback", handlers.PublicCreateFeedback(db, feedbackNotifier))
	g.GET("/icon/:appID", handlers.AppIcon(db))
	g.GET("/icon/products/:productID", handlers.ProductIcon(db))

	g.GET("/ios/:releaseID/manifest.plist", handlers.IOSManifest(db, cfg))
	g.GET("/d/:releaseID", handlers.Download(db))

	// Public visit counting (client-side)
	g.POST("/events", handlers.PostEvent(db))

	// UDID endpoints
	g.GET("/udid/profile.mobileconfig", handlers.UDIDProfile(db, cfg))
	g.POST("/udid/callback", handlers.UDIDCallback(db, cfg))
	g.GET("/udid/status", handlers.UDIDStatus(db))

	// Upload (auth required, with CORS for cross-domain uploads)
	upload := g.Group("/")
	upload.Use(func(c *gin.Context) {
		// Allow cross-origin uploads when upload_domain is configured
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Auth-Token")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	upload.OPTIONS("/upload", func(c *gin.Context) { c.Status(204) })
	upload.POST("/upload", middleware.RequireToken(cfg, "upload"), handlers.Upload(db, cfg))
	upload.OPTIONS("/smart-upload", func(c *gin.Context) { c.Status(204) })
	upload.POST("/smart-upload", middleware.RequireToken(cfg, "upload"), handlers.SmartUpload(db, cfg))
	upload.OPTIONS("/"+adminPath+"/api/parse-app", func(c *gin.Context) { c.Status(204) })
	upload.POST("/"+adminPath+"/api/parse-app", middleware.RequireToken(cfg, "admin"), handlers.ParseAppInfo())
	upload.OPTIONS("/"+adminPath+"/api/releases/:releaseID/replace-file", func(c *gin.Context) { c.Status(204) })
	upload.POST("/"+adminPath+"/api/releases/:releaseID/replace-file", middleware.RequireToken(cfg, "admin"), handlers.AdminReplaceReleaseFile(db, cfg))

	// Admin APIs (under configurable admin path)
	admin := g.Group("/" + adminPath)
	admin.Use(middleware.RequireToken(cfg, "admin"))
	// JSON
	admin.GET("/api/ios_devices", handlers.AdminListIOSDevices(db))
	admin.GET("/api/ios_variants", handlers.AdminListIOSVariants(db))
	// Exports

	// List apps
	admin.GET("/api/apps", handlers.AdminListApps(db))
	// Get single app with releases and provisioning profiles
	admin.GET("/api/apps/:appID", handlers.AdminGetApp(db))
	admin.GET("/api/products", handlers.AdminListProducts(db))
	admin.POST("/api/products", handlers.AdminCreateProduct(db))
	admin.GET("/api/products/:productID", handlers.AdminGetProduct(db))
	admin.GET("/api/products/:productID/announcements", handlers.AdminListAnnouncements(db))
	admin.POST("/api/products/:productID/announcements", handlers.AdminCreateAnnouncement(db))
	admin.GET("/api/feedback", handlers.AdminListFeedback(db))
	admin.PATCH("/api/feedback/:feedbackID", handlers.AdminUpdateFeedback(db))
	admin.DELETE("/api/feedback/:feedbackID", handlers.AdminDeleteFeedback(db))
	admin.GET("/api/announcements/:announcementID", handlers.AdminGetAnnouncement(db))
	admin.PUT("/api/announcements/:announcementID", handlers.AdminUpdateAnnouncement(db))
	admin.DELETE("/api/announcements/:announcementID", handlers.AdminDeleteAnnouncement(db))
	admin.PUT("/api/products/:productID", handlers.AdminUpdateProduct(db))
	admin.POST("/api/products/:productID/variants", handlers.AdminCreateVariant(db))
	admin.GET("/api/variants/:variantID/stats", handlers.AdminGetVariantStats(db))
	admin.PUT("/api/variants/:variantID", handlers.AdminUpdateVariant(db))
	admin.DELETE("/api/products/:productID", handlers.AdminDeleteProduct(db))
	admin.DELETE("/api/variants/:variantID", handlers.AdminDeleteVariant(db))
	admin.PUT("/api/releases/:releaseID", handlers.AdminUpdateRelease(db))
	admin.DELETE("/api/releases/:releaseID", handlers.AdminDeleteRelease(db))
	admin.GET("/api/releases/:releaseID/file", handlers.AdminDownloadRelease(db))

	// App publish/unpublish
	admin.PUT("/api/apps/:appID/publish", handlers.PublishApp(db))
	admin.PUT("/api/apps/:appID/unpublish", handlers.UnpublishApp(db))

	// System settings
	admin.GET("/api/settings", handlers.GetSystemSettings(db))
	admin.PUT("/api/settings", handlers.UpdateSystemSettings(db))
	admin.POST("/api/settings/email/test", handlers.TestEmailSettings(db))

	// Upload config (for frontend to know upload endpoint)
	admin.GET("/api/upload-config", handlers.GetUploadConfig(db))

	// Apple Developer Portal integration
	admin.GET("/api/apple/status", handlers.AppleStatus(db))
	admin.GET("/api/apple/devices", handlers.AppleListDevices(db))
	admin.POST("/api/devices/:id/register-apple", handlers.RegisterDeviceToApple(db))
	admin.POST("/api/devices/register-apple", handlers.BatchRegisterDevicesToApple(db))

	admin.GET("/exports/releases.csv", handlers.ExportReleases(db))
	admin.GET("/exports/ios_devices.csv", handlers.ExportIOSDevices(db))

	// Root landing: project intro page (front SPA intro view)
	g.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "app.html", gin.H{"Title": "Fenfa", "Message": ""})
	})

	// Admin HTML page (public page; exports under /<adminPath>/exports require token)
	g.GET("/"+adminPath, func(c *gin.Context) {
		if cfg.Server.DevProxyAdmin != "" {
			target := cfg.Server.DevProxyAdmin + "/"
			c.Redirect(http.StatusTemporaryRedirect, target)
			return
		}
		c.HTML(http.StatusOK, "admin.html", gin.H{"Title": "Modern distribution", "AdminPath": adminPath})
	})

	// Fallback 404 page for unmatched routes
	g.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.html", gin.H{"Title": "404 · Not Found"})
	})

	return nil
}
