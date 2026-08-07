package store

import (
	"testing"
)

func TestAutoMigrateUpgradesLegacyReleaseSchema(t *testing.T) {
	db := openTestDB(t)

	if err := db.Exec(`
		CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			platform TEXT,
			bundle_id TEXT,
			application_id TEXT,
			name TEXT,
			icon_path TEXT,
			published INTEGER DEFAULT 1,
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create legacy apps table: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE releases (
			id TEXT PRIMARY KEY,
			app_id TEXT,
			variant_id TEXT,
			version TEXT,
			build INTEGER,
			changelog TEXT,
			min_os TEXT,
			size_bytes INTEGER,
			sha256 TEXT,
			storage_path TEXT,
			file_name TEXT,
			file_ext TEXT,
			mime_type TEXT,
			download_count INTEGER,
			channel TEXT,
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create legacy releases table: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO releases (id, app_id, variant_id, version, build, created_at)
		VALUES ('rel_legacy_schema', '', 'var_legacy', '1.0.0', 1, '2026-01-01T00:00:00Z')
	`).Error; err != nil {
		t.Fatalf("insert legacy release: %v", err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate legacy schema: %v", err)
	}
	if !db.Migrator().HasColumn(&Release{}, "force_update") {
		t.Fatalf("expected releases.force_update column to exist after migration")
	}
	if !db.Migrator().HasTable("announcements") {
		t.Fatalf("expected announcements table to exist after migration")
	}

	var forceUpdate int
	if err := db.Raw("SELECT force_update FROM releases WHERE id = ?", "rel_legacy_schema").Scan(&forceUpdate).Error; err != nil {
		t.Fatalf("read legacy force_update: %v", err)
	}
	if forceUpdate != 0 {
		t.Fatalf("expected legacy release force_update=false, got %d", forceUpdate)
	}

	var row struct {
		ID      string
		Version string
	}
	if err := db.Raw("SELECT id, version FROM releases WHERE id = ?", "rel_legacy_schema").Scan(&row).Error; err != nil {
		t.Fatalf("read legacy row: %v", err)
	}
	if row.ID != "rel_legacy_schema" || row.Version != "1.0.0" {
		t.Fatalf("expected legacy release data to remain, got %+v", row)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}
	if !db.Migrator().HasColumn(&Release{}, "force_update") {
		t.Fatalf("expected force_update column after second migrate")
	}
}

func TestAutoMigrateCreatesAnnouncementTableAndDefaultsEnabled(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if !db.Migrator().HasTable("announcements") {
		t.Fatalf("expected announcements table to exist")
	}

	product := Product{ID: "prd_ann_migration", Slug: "ann-migration", Name: "Ann Migration"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	announcement := Announcement{
		ID:        "ann_migration",
		ProductID: product.ID,
		Title:     "Hello",
		Content:   "World",
	}
	if err := db.Create(&announcement).Error; err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if !announcement.Enabled {
		t.Fatalf("expected announcement enabled default true, got false")
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}
}
