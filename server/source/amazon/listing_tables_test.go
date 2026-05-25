package amazon

import (
	"context"
	"path/filepath"
	"testing"

	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitAmazonTablesMigratesAllBusinessModels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "amazon-init.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	initializer := &initAmazonTables{}
	ctx := context.WithValue(context.Background(), "db", db)

	if initializer.TableCreated(ctx) {
		t.Fatal("fresh database unexpectedly reports Amazon tables as created")
	}

	if _, err := initializer.MigrateTable(ctx); err != nil {
		t.Fatalf("migrate Amazon tables: %v", err)
	}

	if !initializer.TableCreated(ctx) {
		t.Fatal("Amazon table check did not pass after migration")
	}

	for _, model := range amazonModel.BusinessModels() {
		if !db.Migrator().HasTable(model) {
			t.Fatalf("missing migrated table for %T", model)
		}
	}
}
