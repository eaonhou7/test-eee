package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate(amazonModel.BusinessModels()...)
	if err != nil {
		return err
	}
	return syncAmazonSeeds(db)
}
