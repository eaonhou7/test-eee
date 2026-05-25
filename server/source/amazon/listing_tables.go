package amazon

import (
	"context"

	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	"gorm.io/gorm"
)

const initOrderAmazonTables = system.InitOrderInternal + 100

type initAmazonTables struct{}

func init() {
	system.RegisterInit(initOrderAmazonTables, &initAmazonTables{})
}

func (i *initAmazonTables) InitializerName() string {
	return "amazon_business_tables"
}

func (i *initAmazonTables) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	return ctx, db.AutoMigrate(amazonModel.BusinessModels()...)
}

func (i *initAmazonTables) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	for _, model := range amazonModel.BusinessModels() {
		if !db.Migrator().HasTable(model) {
			return false
		}
	}
	return true
}

func (i *initAmazonTables) InitializeData(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (i *initAmazonTables) DataInserted(ctx context.Context) bool {
	return true
}
