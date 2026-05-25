package amazon

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	exampleModel "github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestCollectorUpsertDetailCreatesProductImagesAndMaterialRecords(t *testing.T) {
	setupCollectorTestDB(t)

	service := CollectorService{}
	price := 29.99
	rating := 4.6
	reviewCount := 128
	result, err := service.UpsertDetail(context.Background(), amazonReq.CollectedProductUpsertFromExtensionReq{
		SiteCode:         "US",
		ASIN:             "B012345678",
		Title:            "Desk Lamp",
		ProductURL:       "https://www.amazon.com/dp/B012345678",
		PriceAmount:      &price,
		CurrencyCode:     "USD",
		RatingValue:      &rating,
		ReviewCount:      &reviewCount,
		MainImageURL:     "https://images.example.com/main.jpg",
		GalleryImageURLs: []string{"https://images.example.com/main.jpg", "https://images.example.com/extra.jpg"},
		BulletPoints:     []string{"Warm light"},
	})
	if err != nil {
		t.Fatalf("upsert detail: %v", err)
	}
	if result.ID == 0 {
		t.Fatal("expected collected product id")
	}
	if result.MainImageFileID == nil {
		t.Fatal("expected main image file id")
	}
	if len(result.Images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(result.Images))
	}
	if result.CollectStatus != collectorStatusSuccess {
		t.Fatalf("expected collect status success, got %s", result.CollectStatus)
	}

	var productCount int64
	if err := global.GVA_DB.Model(&amazonModel.CollectedProduct{}).Count(&productCount).Error; err != nil {
		t.Fatalf("count products: %v", err)
	}
	if productCount != 1 {
		t.Fatalf("expected 1 collected product, got %d", productCount)
	}

	var fileCount int64
	if err := global.GVA_DB.Model(&exampleModel.ExaFileUploadAndDownload{}).Count(&fileCount).Error; err != nil {
		t.Fatalf("count files: %v", err)
	}
	if fileCount != 2 {
		t.Fatalf("expected 2 material records, got %d", fileCount)
	}
}

func TestCollectorUpsertDetailReusesSiteAndASINOnRepeatedCollect(t *testing.T) {
	setupCollectorTestDB(t)

	service := CollectorService{}
	price := 19.9
	first, err := service.UpsertDetail(context.Background(), amazonReq.CollectedProductUpsertFromExtensionReq{
		SiteCode:         "CA",
		ASIN:             "B0TEST0001",
		Title:            "Original Title",
		ProductURL:       "https://www.amazon.ca/dp/B0TEST0001",
		PriceAmount:      &price,
		CurrencyCode:     "CAD",
		MainImageURL:     "https://images.example.com/a.jpg",
		GalleryImageURLs: []string{"https://images.example.com/b.jpg"},
	})
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	updatedPrice := 21.5
	second, err := service.UpsertDetail(context.Background(), amazonReq.CollectedProductUpsertFromExtensionReq{
		SiteCode:         "CA",
		ASIN:             "B0TEST0001",
		Title:            "Updated Title",
		ProductURL:       "https://www.amazon.ca/dp/B0TEST0001",
		PriceAmount:      &updatedPrice,
		CurrencyCode:     "CAD",
		MainImageURL:     "https://images.example.com/a.jpg",
		GalleryImageURLs: []string{"https://images.example.com/b.jpg"},
	})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected repeated collect to reuse product id, got %d and %d", first.ID, second.ID)
	}
	if second.Title != "Updated Title" {
		t.Fatalf("expected title to update, got %s", second.Title)
	}

	var productCount int64
	if err := global.GVA_DB.Model(&amazonModel.CollectedProduct{}).Count(&productCount).Error; err != nil {
		t.Fatalf("count products: %v", err)
	}
	if productCount != 1 {
		t.Fatalf("expected 1 collected product after repeated collect, got %d", productCount)
	}

	var fileCount int64
	if err := global.GVA_DB.Model(&exampleModel.ExaFileUploadAndDownload{}).Count(&fileCount).Error; err != nil {
		t.Fatalf("count files: %v", err)
	}
	if fileCount != 2 {
		t.Fatalf("expected existing material records to be reused, got %d", fileCount)
	}
}

func setupCollectorTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "collector.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&amazonModel.CollectedProduct{}, &amazonModel.CollectedProductImage{}, &exampleModel.ExaFileUploadAndDownload{}); err != nil {
		t.Fatalf("migrate collector tables: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
}
