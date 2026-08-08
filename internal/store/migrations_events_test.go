package store

import "testing"

func TestAutoMigrateBackfillsVisitCountAndDropsLegacyEvents(t *testing.T) {
	db := openTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("initial auto migrate: %v", err)
	}

	variantA := Variant{
		ID:         "var_legacy_a",
		ProductID:  "prd_legacy_a",
		Platform:   "windows",
		Identifier: "com.example.a",
	}
	variantB := Variant{
		ID:         "var_legacy_b",
		ProductID:  "prd_legacy_b",
		Platform:   "ios",
		Identifier: "com.example.b",
	}
	if err := db.Create(&variantA).Error; err != nil {
		t.Fatalf("create variant a: %v", err)
	}
	if err := db.Create(&variantB).Error; err != nil {
		t.Fatalf("create variant b: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ts DATETIME,
			type TEXT,
			app_id TEXT,
			variant_id TEXT,
			release_id TEXT,
			ip TEXT,
			ua TEXT,
			extra TEXT
		)
	`).Error; err != nil {
		t.Fatalf("create legacy events table: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO events (type, variant_id, release_id) VALUES
			('visit', 'var_legacy_a', 'rel_a_1'),
			('visit', 'var_legacy_a', 'rel_a_2'),
			('visit', 'var_legacy_b', 'rel_b_1'),
			('click', 'var_legacy_a', 'rel_a_2'),
			('visit', '', 'rel_unowned')
	`).Error; err != nil {
		t.Fatalf("insert legacy events: %v", err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate legacy events: %v", err)
	}

	var gotA Variant
	if err := db.Where("id = ?", variantA.ID).First(&gotA).Error; err != nil {
		t.Fatalf("load variant a: %v", err)
	}
	if gotA.VisitCount != 2 {
		t.Fatalf("expected variant a visit_count 2, got %d", gotA.VisitCount)
	}

	var gotB Variant
	if err := db.Where("id = ?", variantB.ID).First(&gotB).Error; err != nil {
		t.Fatalf("load variant b: %v", err)
	}
	if gotB.VisitCount != 1 {
		t.Fatalf("expected variant b visit_count 1, got %d", gotB.VisitCount)
	}

	if db.Migrator().HasTable("events") {
		t.Fatalf("expected legacy events table to be dropped")
	}
}
