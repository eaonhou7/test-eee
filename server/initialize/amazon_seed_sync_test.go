package initialize

import (
	"path/filepath"
	"testing"

	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	sysModel "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSyncAmazonSeedsSkipsSystemSeedsBeforeInitData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "amazon-seeds.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	models := []interface{}{
		&sysModel.SysUser{},
		&sysModel.SysAuthority{},
		&sysModel.SysBaseMenu{},
		&sysModel.SysApi{},
	}
	models = append(models, amazonModel.BusinessModels()...)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate models: %v", err)
	}

	if err := syncAmazonSeeds(db); err != nil {
		t.Fatalf("sync amazon seeds: %v", err)
	}

	var menuCount int64
	if err := db.Model(&sysModel.SysBaseMenu{}).Count(&menuCount).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if menuCount != 0 {
		t.Fatalf("expected no menus before system init data, got %d", menuCount)
	}

	var apiCount int64
	if err := db.Model(&sysModel.SysApi{}).Count(&apiCount).Error; err != nil {
		t.Fatalf("count apis: %v", err)
	}
	if apiCount != 0 {
		t.Fatalf("expected no apis before system init data, got %d", apiCount)
	}
}
