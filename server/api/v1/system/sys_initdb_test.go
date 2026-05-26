package system

import (
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseNeedsInitWhenConfiguredDatabaseHasNoAdminUser(t *testing.T) {
	previousDB := global.GVA_DB
	t.Cleanup(func() {
		global.GVA_DB = previousDB
	})

	global.GVA_DB = nil
	if !databaseNeedsInit() {
		t.Fatal("nil database should require initialization")
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "init-check.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	global.GVA_DB = db

	if !databaseNeedsInit() {
		t.Fatal("database without sys_users table should require initialization")
	}

	if err := db.AutoMigrate(&sysModel.SysUser{}); err != nil {
		t.Fatalf("migrate sys_users: %v", err)
	}
	if !databaseNeedsInit() {
		t.Fatal("database without admin user should require initialization")
	}

	if err := db.Create(&sysModel.SysUser{Username: "admin", Password: "test"}).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if databaseNeedsInit() {
		t.Fatal("database with admin user should not require initialization")
	}
}
