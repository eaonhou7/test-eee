package amazon

import (
	"path/filepath"
	"testing"

	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeUniqueLocalePayloadsDeduplicatesSameLocale(t *testing.T) {
	locales := normalizeUniqueLocalePayloads([]amazonReq.ListingLocalePayloadDTO{
		{
			ID:                  1,
			LocaleCode:          "en_US",
			ItemName:            "Desk Lamp",
			LocalizedAttributes: commonModel.JSONMap{"material": "metal"},
		},
		{
			ID:                 2,
			LocaleCode:         "en-US",
			BulletPoints:       []string{"Warm light"},
			ProductDescription: "Long description",
			SearchTerms:        []string{"lamp"},
		},
		{
			LocaleCode: "fr_CA",
			ItemName:   "Lampe",
		},
	})

	if len(locales) != 2 {
		t.Fatalf("expected 2 unique locales, got %d", len(locales))
	}
	if locales[0].LocaleCode != "en_US" {
		t.Fatalf("expected normalized locale code en_US, got %s", locales[0].LocaleCode)
	}
	if locales[0].ItemName != "Desk Lamp" {
		t.Fatalf("expected original item name to be kept, got %s", locales[0].ItemName)
	}
	if len(locales[0].BulletPoints) != 1 || locales[0].BulletPoints[0] != "Warm light" {
		t.Fatalf("expected duplicate locale bullet points to be merged, got %#v", locales[0].BulletPoints)
	}
	if locales[0].ProductDescription != "Long description" {
		t.Fatalf("expected duplicate locale description to be merged, got %s", locales[0].ProductDescription)
	}
	if len(locales[0].SearchTerms) != 1 || locales[0].SearchTerms[0] != "lamp" {
		t.Fatalf("expected duplicate locale search terms to be merged, got %#v", locales[0].SearchTerms)
	}
	if locales[1].LocaleCode != "fr_CA" {
		t.Fatalf("expected second locale to remain fr_CA, got %s", locales[1].LocaleCode)
	}
}

func TestReplaceLocalesCanReplaceSameLocale(t *testing.T) {
	db := newListingItemServiceTestDB(t, &amazonModel.ListingItemLocale{})

	if err := replaceLocales(db, 4, []amazonReq.ListingLocalePayloadDTO{
		{LocaleCode: "en_US", ItemName: "Old title"},
	}); err != nil {
		t.Fatalf("first replace locales: %v", err)
	}
	if err := replaceLocales(db, 4, []amazonReq.ListingLocalePayloadDTO{
		{LocaleCode: "en_US", ItemName: "New title"},
	}); err != nil {
		t.Fatalf("second replace locales: %v", err)
	}

	var locales []amazonModel.ListingItemLocale
	if err := db.Unscoped().Where("item_marketplace_id = ?", 4).Find(&locales).Error; err != nil {
		t.Fatalf("find locales: %v", err)
	}
	if len(locales) != 1 {
		t.Fatalf("expected one physical locale row after replacement, got %d", len(locales))
	}
	if locales[0].ItemName != "New title" {
		t.Fatalf("expected latest locale title, got %s", locales[0].ItemName)
	}
}

func TestReplaceListingImagesCanReplaceSameSlot(t *testing.T) {
	db := newListingItemServiceTestDB(t, &amazonModel.ListingItemImage{})
	marketplaceID := uint(7)

	if err := replaceListingImages(db, 3, &marketplaceID, []amazonReq.ListingImageAssetDTO{
		{SlotCode: "MAIN", FileID: 1, ImageURL: "https://example.test/old.jpg"},
	}); err != nil {
		t.Fatalf("first replace images: %v", err)
	}
	if err := replaceListingImages(db, 3, &marketplaceID, []amazonReq.ListingImageAssetDTO{
		{SlotCode: "MAIN", FileID: 2, ImageURL: "https://example.test/new.jpg"},
	}); err != nil {
		t.Fatalf("second replace images: %v", err)
	}

	var images []amazonModel.ListingItemImage
	if err := db.Unscoped().Where("item_id = ? AND item_marketplace_id = ?", 3, marketplaceID).Find(&images).Error; err != nil {
		t.Fatalf("find images: %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("expected one physical image row after replacement, got %d", len(images))
	}
	if images[0].ImageURL != "https://example.test/new.jpg" {
		t.Fatalf("expected latest image url, got %s", images[0].ImageURL)
	}
}

func newListingItemServiceTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "listing-item.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
