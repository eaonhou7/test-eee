package amazon

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	exampleModel "github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestCollector1688CreateTaskReturnsSearchURLAndSystemCode(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-001")

	service := Collector1688Service{}
	result, err := service.CreateTask(context.Background(), amazonReq.Create1688CollectTaskReq{
		ListingItemID: item.ID,
		MainImageURL:  "https://img.alicdn.com/example-main.jpg",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if result.TaskID == 0 || result.TaskToken == "" {
		t.Fatal("expected task id and token")
	}
	if result.SystemCode != item.SKU {
		t.Fatalf("expected system code %s, got %s", item.SKU, result.SystemCode)
	}
	if !strings.Contains(result.SearchURL, "__gva1688Task=") {
		t.Fatalf("expected search url with task token, got %s", result.SearchURL)
	}
	if !strings.HasPrefix(result.SearchURL, "https://s.1688.com/shen/sell_offer.html?") {
		t.Fatalf("expected search url to use current 1688 image search entry, got %s", result.SearchURL)
	}
	parsedURL, err := url.Parse(result.SearchURL)
	if err != nil {
		t.Fatalf("parse search url: %v", err)
	}
	query := parsedURL.Query()
	if query.Get("tab") != "imageSearch" {
		t.Fatalf("expected image search tab, got %s in %s", query.Get("tab"), result.SearchURL)
	}
	if query.Get("__gva1688Task") != result.TaskToken {
		t.Fatalf("expected task token in search url, got %s in %s", query.Get("__gva1688Task"), result.SearchURL)
	}
	if query.Get("__gva1688Image") != "https://img.alicdn.com/example-main.jpg" {
		t.Fatalf("expected source image param in search url, got %s in %s", query.Get("__gva1688Image"), result.SearchURL)
	}
	if query.Get("imageAddress") != "" {
		t.Fatalf("expected search url not to use native imageAddress for external image, got %s", result.SearchURL)
	}
	tabIndex := strings.Index(result.SearchURL, "?tab=imageSearch")
	taskIndex := strings.Index(result.SearchURL, "&__gva1688Task=")
	imageIndex := strings.Index(result.SearchURL, "&__gva1688Image=")
	if tabIndex < 0 || taskIndex < 0 || imageIndex < 0 || !(tabIndex < taskIndex && taskIndex < imageIndex) {
		t.Fatalf("expected 1688 image search url param order tab/task/source-image, got %s", result.SearchURL)
	}
}

func TestCollector1688CreateTaskRejectsMissingListingItem(t *testing.T) {
	setupCollector1688TestDB(t)
	service := Collector1688Service{}
	if _, err := service.CreateTask(context.Background(), amazonReq.Create1688CollectTaskReq{
		ListingItemID: 999,
		MainImageURL:  "https://img.alicdn.com/example-main.jpg",
	}); err == nil {
		t.Fatal("expected create task to fail for missing listing item")
	}
}

func TestCollector1688CreateRepairTaskOpensExistingOfferDetail(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-REPAIR-TASK")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	upserted, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700099",
		Title:        "待修复商品",
		ProductURL:   "https://detail.1688.com/offer/700099.html",
		MainImageURL: "https://img.alicdn.com/repair-task-main.jpg",
		PriceText:    "¥ 5.80",
		SKUOffers:    validCollector1688SKUOffers(),
	})
	if err != nil {
		t.Fatalf("upsert detail: %v", err)
	}

	repairTask, err := service.CreateRepairTask(context.Background(), amazonReq.Create1688RepairTaskReq{
		CollectedProductID: upserted.CollectedProductID,
	})
	if err != nil {
		t.Fatalf("create repair task: %v", err)
	}
	if repairTask.TaskType != collect1688TaskTypeRepair {
		t.Fatalf("expected repair task type, got %s", repairTask.TaskType)
	}
	if !strings.Contains(repairTask.DetailURL, "__gva1688Task=") || !strings.Contains(repairTask.DetailURL, "/offer/700099.html") {
		t.Fatalf("expected repair detail URL with token, got %s", repairTask.DetailURL)
	}
	if repairTask.CollectedProductID != upserted.CollectedProductID {
		t.Fatalf("expected repair task to target product %d, got %d", upserted.CollectedProductID, repairTask.CollectedProductID)
	}
}

func TestCollector1688UpsertDetailCreatesProductImagesBindingsAndSuccessTask(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-002")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	result, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:        task.TaskToken,
		OfferID:          "700001",
		Title:            "1688 Test Product",
		ProductURL:       "https://detail.1688.com/offer/700001.html",
		MainImageURL:     "https://img.alicdn.com/example-main.jpg",
		GalleryImageURLs: []string{"https://img.alicdn.com/example-gallery.jpg"},
		DetailImageURLs:  []string{"https://img.alicdn.com/example-detail.jpg"},
		PriceText:        "¥ 12.50 - 18.80",
		CurrencyCode:     "CNY",
		OrderUnit:        "件",
		SellerCompany:    "测试公司",
		ShopName:         "测试店铺",
		CategoryPath:     []string{"家居百货", "收纳整理", "收纳盒"},
		SKUOffers:        validCollector1688SKUOffers(),
	})
	if err != nil {
		t.Fatalf("upsert detail: %v", err)
	}
	if result.CollectedProductID == 0 {
		t.Fatal("expected collected product id")
	}

	var productCount int64
	if err := global.GVA_DB.Model(&amazonModel.Collected1688Product{}).Count(&productCount).Error; err != nil {
		t.Fatalf("count products: %v", err)
	}
	if productCount != 1 {
		t.Fatalf("expected 1 collected 1688 product, got %d", productCount)
	}

	var imageCount int64
	if err := global.GVA_DB.Model(&amazonModel.Collected1688ProductImage{}).Count(&imageCount).Error; err != nil {
		t.Fatalf("count images: %v", err)
	}
	if imageCount != 3 {
		t.Fatalf("expected 3 collected images, got %d", imageCount)
	}

	var binding amazonModel.Collect1688Binding
	if err := global.GVA_DB.Where("listing_item_id = ?", item.ID).First(&binding).Error; err != nil {
		t.Fatalf("find binding: %v", err)
	}
	if !binding.IsActive {
		t.Fatal("expected active binding")
	}

	var storedTask amazonModel.Collect1688Task
	if err := global.GVA_DB.First(&storedTask, task.TaskID).Error; err != nil {
		t.Fatalf("find task: %v", err)
	}
	if storedTask.Status != collect1688TaskStatusSuccess {
		t.Fatalf("expected success task status, got %s", storedTask.Status)
	}
}

func TestCollector1688UpsertDetailStoresExtendedFields(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-EXTENDED")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	result, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700120",
		Title:        "扩展字段商品",
		ProductURL:   "https://detail.1688.com/offer/700120.html",
		MainImageURL: "https://img.alicdn.com/extended-main.jpg",
		PriceText:    "¥ 5.80",
		SKUOffers:    validCollector1688SKUOffers(),
		ProductAttributes: commonModel.JSONMap{
			"材质": "PVC",
			"重量": "0.2kg",
		},
		PackageInfo: commonModel.JSONMap{
			"weightKg": 0.35,
			"lengthCm": 12,
			"widthCm":  8,
			"heightCm": 5,
			"rawText":  "包装尺寸 12*8*5cm，重量 0.35kg",
		},
		DetailSections: []commonModel.JSONMap{{
			"title": "商品详情",
			"text":  "详情文字",
			"imageUrls": []string{
				"https://img.alicdn.com/extended-detail.jpg",
			},
		}},
		DetailText: "详情文字",
	})
	if err != nil {
		t.Fatalf("upsert detail: %v", err)
	}

	detail, err := service.Find(context.Background(), result.CollectedProductID)
	if err != nil {
		t.Fatalf("find detail: %v", err)
	}
	if detail.ProductAttributes["材质"] != "PVC" {
		t.Fatalf("expected product attributes to be returned, got %#v", detail.ProductAttributes)
	}
	if detail.PackageInfo["rawText"] != "包装尺寸 12*8*5cm，重量 0.35kg" {
		t.Fatalf("expected package info to be returned, got %#v", detail.PackageInfo)
	}
	if len(detail.DetailSections) != 1 || detail.DetailSections[0]["title"] != "商品详情" {
		t.Fatalf("expected detail sections to be returned, got %#v", detail.DetailSections)
	}
	if detail.DetailText != "详情文字" {
		t.Fatalf("expected detail text, got %s", detail.DetailText)
	}
}

func TestCollector1688UpsertDetailRejectsMissingSKUOffers(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-MISSING-SKU")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700901",
		Title:        "缺少 SKU 报价",
		ProductURL:   "https://detail.1688.com/offer/700901.html",
		MainImageURL: "https://img.alicdn.com/missing-sku.jpg",
		PriceText:    "¥ 5.80",
	}); err == nil || !strings.Contains(err.Error(), "skuOffers") {
		t.Fatalf("expected skuOffers validation error, got %v", err)
	}

	var productCount int64
	if err := global.GVA_DB.Model(&amazonModel.Collected1688Product{}).Count(&productCount).Error; err != nil {
		t.Fatalf("count products: %v", err)
	}
	if productCount != 0 {
		t.Fatalf("expected no product to be created, got %d", productCount)
	}
}

func TestCollector1688UpsertDetailRejectsEmptySKUOfferRows(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-EMPTY-SKU")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700903",
		Title:        "空 SKU 报价行",
		ProductURL:   "https://detail.1688.com/offer/700903.html",
		MainImageURL: "https://img.alicdn.com/empty-sku.jpg",
		PriceText:    "¥ 5.80",
		SKUOffers:    []commonModel.JSONMap{{}},
	}); err == nil || !strings.Contains(err.Error(), "skuOffers") {
		t.Fatalf("expected skuOffers validation error, got %v", err)
	}
}

func TestCollector1688UpsertDetailRejectsMissingMainImage(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-MISSING-IMAGE")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:  task.TaskToken,
		OfferID:    "700902",
		Title:      "缺少主图",
		ProductURL: "https://detail.1688.com/offer/700902.html",
		PriceText:  "¥ 5.80",
		SKUOffers:  validCollector1688SKUOffers(),
	}); err == nil || !strings.Contains(err.Error(), "mainImageUrl") {
		t.Fatalf("expected mainImageUrl validation error, got %v", err)
	}

	var productCount int64
	if err := global.GVA_DB.Model(&amazonModel.Collected1688Product{}).Count(&productCount).Error; err != nil {
		t.Fatalf("count products: %v", err)
	}
	if productCount != 0 {
		t.Fatalf("expected no product to be created, got %d", productCount)
	}
}

func TestCollector1688UpsertDetailRejectsMainImageDownloadFailure(t *testing.T) {
	setupCollector1688TestDB(t)
	setCollector1688ImageHTTPClient(t, &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("forbidden")),
			Request:    req,
		}, nil
	})})
	item := seedCollector1688ListingItem(t, "SKU-1688-BAD-MAIN")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700914",
		Title:        "主图下载失败",
		ProductURL:   "https://detail.1688.com/offer/700914.html",
		MainImageURL: "https://img.alicdn.com/bad-main.jpg",
		PriceText:    "¥ 5.80",
		SKUOffers:    validCollector1688SKUOffers(),
	}); err == nil || !strings.Contains(err.Error(), "主图入库失败") {
		t.Fatalf("expected main image material error, got %v", err)
	}

	var productCount int64
	if err := global.GVA_DB.Model(&amazonModel.Collected1688Product{}).Count(&productCount).Error; err != nil {
		t.Fatalf("count products: %v", err)
	}
	if productCount != 0 {
		t.Fatalf("expected no product to be created, got %d", productCount)
	}
}

func TestCollector1688UpsertDetailCachesImagesAndDoesNotExpose1688ExternalURL(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-CACHE")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	result, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:        task.TaskToken,
		OfferID:          "700915",
		Title:            "图片缓存商品",
		ProductURL:       "https://detail.1688.com/offer/700915.html",
		MainImageURL:     "https://img.alicdn.com/cache-main.jpg",
		GalleryImageURLs: []string{"https://img.alicdn.com/cache-gallery.jpg"},
		PriceText:        "¥ 5.80",
		SKUOffers: []commonModel.JSONMap{{
			"skuKey":        "pink",
			"attributeText": "粉色",
			"priceText":     "¥5.80",
			"imageUrl":      "https://img.alicdn.com/cache-sku.jpg",
		}},
	})
	if err != nil {
		t.Fatalf("upsert detail: %v", err)
	}
	detail, err := service.Find(context.Background(), result.CollectedProductID)
	if err != nil {
		t.Fatalf("find detail: %v", err)
	}
	if detail.MainImageURL == "" || strings.Contains(detail.MainImageURL, "alicdn.com") {
		t.Fatalf("expected cached main image URL, got %s", detail.MainImageURL)
	}
	if len(detail.Images) != 2 || detail.Images[0].File == nil || strings.Contains(detail.Images[0].File.URL, "alicdn.com") {
		t.Fatalf("expected image file cache in detail, got %#v", detail.Images)
	}
	if len(detail.SKUOffers) != 1 || strings.Contains(formatJsonValueForTest(detail.SKUOffers[0]["imageUrl"]), "alicdn.com") {
		t.Fatalf("expected localized sku image URL, got %#v", detail.SKUOffers)
	}
	if formatJsonValueForTest(detail.SKUOffers[0]["originalImageUrl"]) == "" {
		t.Fatalf("expected original SKU image URL to be preserved, got %#v", detail.SKUOffers[0])
	}

	var images []amazonModel.Collected1688ProductImage
	if err := global.GVA_DB.Where("collected_product_id = ?", result.CollectedProductID).Order("sort ASC").Find(&images).Error; err != nil {
		t.Fatalf("find images: %v", err)
	}
	if len(images) != 2 || !images[0].IsMain || images[0].OriginalURL != "https://img.alicdn.com/cache-main.jpg" {
		t.Fatalf("expected only mainImageUrl to be marked as main, got %#v", images)
	}
}

func TestCollector1688InferFulfillmentProfileUsesPackageInfo(t *testing.T) {
	product := amazonModel.Collected1688Product{
		PackageInfoJSON: encodeJSONObject(commonModel.JSONMap{
			"weightKg": 0.42,
			"lengthCm": 14,
			"widthCm":  9,
			"heightCm": 6,
		}),
		ProductAttributesJSON: encodeJSONObject(commonModel.JSONMap{
			"是否带电": "不带电",
		}),
	}

	weight, length, width, height, containsBattery, raw := inferFulfillmentProfile(product)
	if weight == nil || *weight != 0.42 {
		t.Fatalf("expected package weight, got %#v", weight)
	}
	if length == nil || *length != 14 || width == nil || *width != 9 || height == nil || *height != 6 {
		t.Fatalf("expected package dimensions, got length=%#v width=%#v height=%#v", length, width, height)
	}
	if containsBattery == nil || *containsBattery {
		t.Fatalf("expected non-battery package, got %#v", containsBattery)
	}
	if raw["packageInfo"] == nil || raw["productAttributes"] == nil {
		t.Fatalf("expected raw inference to include extended sources, got %#v", raw)
	}
}

func TestCollector1688UpsertDetailWarnsWhenNonMainImageDownloadFails(t *testing.T) {
	setupCollector1688TestDB(t)
	setCollector1688ImageHTTPClient(t, &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.String(), "bad-gallery") {
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("forbidden")),
				Request:    req,
			}, nil
		}
		return fakePNGResponse(req), nil
	})})
	item := seedCollector1688ListingItem(t, "SKU-1688-GALLERY-WARN")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	result, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:        task.TaskToken,
		OfferID:          "700916",
		Title:            "详情图下载告警",
		ProductURL:       "https://detail.1688.com/offer/700916.html",
		MainImageURL:     "https://img.alicdn.com/good-main.jpg",
		GalleryImageURLs: []string{"https://img.alicdn.com/bad-gallery.jpg"},
		PriceText:        "¥ 5.80",
		SKUOffers:        validCollector1688SKUOffers(),
	})
	if err != nil {
		t.Fatalf("upsert detail: %v", err)
	}
	var product amazonModel.Collected1688Product
	if err := global.GVA_DB.First(&product, result.CollectedProductID).Error; err != nil {
		t.Fatalf("find product: %v", err)
	}
	if product.CollectStatus != collectorStatusWarning {
		t.Fatalf("expected warning status, got %s", product.CollectStatus)
	}
	warnings := decodeStringJSON(product.CollectWarningsJSON)
	if len(warnings) == 0 {
		t.Fatal("expected image warning")
	}
}

func TestCollector1688UpsertDetailReusesOfferIDOnRepeatedCollect(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-003")
	service := Collector1688Service{}

	firstTask := mustCreateCollector1688Task(t, service, item.ID)
	first, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    firstTask.TaskToken,
		OfferID:      "700002",
		Title:        "初始标题",
		ProductURL:   "https://detail.1688.com/offer/700002.html",
		MainImageURL: "https://img.alicdn.com/reuse-main.jpg",
		PriceText:    "¥ 9.90",
		SKUOffers:    validCollector1688SKUOffers(),
	})
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	secondTask := mustCreateCollector1688Task(t, service, item.ID)
	second, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    secondTask.TaskToken,
		OfferID:      "700002",
		Title:        "更新标题",
		ProductURL:   "https://detail.1688.com/offer/700002.html",
		MainImageURL: "https://img.alicdn.com/reuse-main.jpg",
		PriceText:    "¥ 12.80",
		SKUOffers:    validCollector1688SKUOffers(),
	})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if first.CollectedProductID != second.CollectedProductID {
		t.Fatalf("expected repeated collect to reuse product id, got %d and %d", first.CollectedProductID, second.CollectedProductID)
	}

	var product amazonModel.Collected1688Product
	if err := global.GVA_DB.First(&product, first.CollectedProductID).Error; err != nil {
		t.Fatalf("find product: %v", err)
	}
	if product.Title != "更新标题" {
		t.Fatalf("expected updated title, got %s", product.Title)
	}
}

func TestCollector1688UpsertDetailAllowsSameTaskSameOfferRefresh(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-REFRESH")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	first, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700910",
		Title:        "刷新前标题",
		ProductURL:   "https://detail.1688.com/offer/700910.html",
		MainImageURL: "https://img.alicdn.com/refresh-before.jpg",
		PriceText:    "¥ 4.99",
		SKUOffers:    validCollector1688SKUOffers(),
	})
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	refreshedOffers := []commonModel.JSONMap{{
		"skuKey":        "blue",
		"attributeText": "蓝色笑脸捏捏球",
		"price":         6.66,
		"priceText":     "¥6.66",
		"stock":         12345,
		"stockText":     "库存12345个",
	}}
	second, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:        task.TaskToken,
		OfferID:          "700910",
		Title:            "刷新后标题",
		ProductURL:       "https://detail.1688.com/offer/700910.html",
		MainImageURL:     "https://img.alicdn.com/refresh-after.jpg",
		GalleryImageURLs: []string{"https://img.alicdn.com/refresh-gallery.jpg"},
		PriceText:        "¥ 6.66",
		SKUOffers:        refreshedOffers,
	})
	if err != nil {
		t.Fatalf("same task refresh: %v", err)
	}
	if first.CollectedProductID != second.CollectedProductID {
		t.Fatalf("expected same product id on refresh, got %d and %d", first.CollectedProductID, second.CollectedProductID)
	}

	var product amazonModel.Collected1688Product
	if err := global.GVA_DB.First(&product, first.CollectedProductID).Error; err != nil {
		t.Fatalf("find product: %v", err)
	}
	if product.Title != "刷新后标题" {
		t.Fatalf("expected refreshed title, got %s", product.Title)
	}
	offers := decodeJSONMapSlice(product.SKUOffersJSON)
	if len(offers) != 1 || offers[0]["priceText"] != "¥6.66" {
		t.Fatalf("expected refreshed sku offers, got %#v", offers)
	}
}

func TestCollector1688UpsertDetailRejectsSameTaskDifferentOfferRefresh(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-REFRESH-DIFF")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700911",
		Title:        "原 offer",
		ProductURL:   "https://detail.1688.com/offer/700911.html",
		MainImageURL: "https://img.alicdn.com/refresh-original.jpg",
		PriceText:    "¥ 4.99",
		SKUOffers:    validCollector1688SKUOffers(),
	}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700912",
		Title:        "另一个 offer",
		ProductURL:   "https://detail.1688.com/offer/700912.html",
		MainImageURL: "https://img.alicdn.com/refresh-other.jpg",
		PriceText:    "¥ 7.77",
		SKUOffers:    validCollector1688SKUOffers(),
	}); err == nil || !strings.Contains(err.Error(), "采集任务已完成") {
		t.Fatalf("expected completed task error for different offer, got %v", err)
	}
}

func TestCollector1688UpsertDetailMarksWarningsAsWarningStatus(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-WARNING")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	result, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:       task.TaskToken,
		OfferID:         "700913",
		Title:           "带非关键告警",
		ProductURL:      "https://detail.1688.com/offer/700913.html",
		MainImageURL:    "https://img.alicdn.com/warning-main.jpg",
		PriceText:       "¥ 8.88",
		SKUOffers:       validCollector1688SKUOffers(),
		CollectWarnings: []string{"详情图未加载完整"},
	})
	if err != nil {
		t.Fatalf("upsert detail: %v", err)
	}

	var product amazonModel.Collected1688Product
	if err := global.GVA_DB.First(&product, result.CollectedProductID).Error; err != nil {
		t.Fatalf("find product: %v", err)
	}
	if product.CollectStatus != collectorStatusWarning {
		t.Fatalf("expected warning status, got %s", product.CollectStatus)
	}
}

func TestCollector1688UpsertDetailSwitchesActiveBindingWhenItemCollectsAnotherOffer(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-004")
	service := Collector1688Service{}

	firstTask := mustCreateCollector1688Task(t, service, item.ID)
	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    firstTask.TaskToken,
		OfferID:      "700003",
		Title:        "第一款",
		ProductURL:   "https://detail.1688.com/offer/700003.html",
		MainImageURL: "https://img.alicdn.com/switch-a.jpg",
		PriceText:    "¥ 11.00",
		SKUOffers:    validCollector1688SKUOffers(),
	}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	secondTask := mustCreateCollector1688Task(t, service, item.ID)
	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    secondTask.TaskToken,
		OfferID:      "700004",
		Title:        "第二款",
		ProductURL:   "https://detail.1688.com/offer/700004.html",
		MainImageURL: "https://img.alicdn.com/switch-b.jpg",
		PriceText:    "¥ 21.00",
		SKUOffers:    validCollector1688SKUOffers(),
	}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var bindings []amazonModel.Collect1688Binding
	if err := global.GVA_DB.Where("listing_item_id = ?", item.ID).Order("id ASC").Find(&bindings).Error; err != nil {
		t.Fatalf("find bindings: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}
	if bindings[0].IsActive {
		t.Fatal("expected first binding to become inactive")
	}
	if !bindings[1].IsActive {
		t.Fatal("expected second binding to become active")
	}
}

func TestCollector1688UpsertDetailRejectsExpiredToken(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-005")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)
	expiredAt := time.Now().Add(-time.Hour)
	if err := global.GVA_DB.Model(&amazonModel.Collect1688Task{}).Where("id = ?", task.TaskID).Update("expires_at", expiredAt).Error; err != nil {
		t.Fatalf("expire task: %v", err)
	}

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700005",
		Title:        "过期任务",
		ProductURL:   "https://detail.1688.com/offer/700005.html",
		MainImageURL: "https://img.alicdn.com/expired.jpg",
		PriceText:    "¥ 8.80",
		SKUOffers:    validCollector1688SKUOffers(),
	}); err == nil {
		t.Fatal("expected expired task to be rejected")
	}
}

func TestCollector1688ReportStateCannotRegressSuccessTask(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-006")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	if _, err := service.UpsertDetail(context.Background(), amazonReq.Collected1688ProductUpsertFromExtensionReq{
		TaskToken:    task.TaskToken,
		OfferID:      "700006",
		Title:        "成功任务",
		ProductURL:   "https://detail.1688.com/offer/700006.html",
		MainImageURL: "https://img.alicdn.com/success.jpg",
		PriceText:    "¥ 5.80",
		SKUOffers:    validCollector1688SKUOffers(),
	}); err != nil {
		t.Fatalf("upsert detail: %v", err)
	}

	if _, err := service.ReportTaskState(context.Background(), amazonReq.Report1688CollectTaskStateReq{
		TaskToken: task.TaskToken,
		Status:    collect1688TaskStatusSearchOpened,
	}); err == nil {
		t.Fatal("expected success task to reject state regression")
	}
}

func TestCollector1688ReportStateSupportsPendingFailure(t *testing.T) {
	setupCollector1688TestDB(t)
	item := seedCollector1688ListingItem(t, "SKU-1688-007")
	service := Collector1688Service{}
	task := mustCreateCollector1688Task(t, service, item.ID)

	result, err := service.ReportTaskState(context.Background(), amazonReq.Report1688CollectTaskStateReq{
		TaskToken:    task.TaskToken,
		Status:       collect1688TaskStatusFailed,
		ErrorMessage: "detail extract failed",
	})
	if err != nil {
		t.Fatalf("report task state: %v", err)
	}
	if result.Status != collect1688TaskStatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
	if result.MainImageURL != task.MainImageURL {
		t.Fatalf("expected task context to include main image URL %s, got %s", task.MainImageURL, result.MainImageURL)
	}
}

func setupCollector1688TestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "collector-1688.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&amazonModel.ListingFamily{},
		&amazonModel.ListingItem{},
		&amazonModel.Collected1688Product{},
		&amazonModel.Collected1688ProductImage{},
		&amazonModel.Collect1688Task{},
		&amazonModel.Collect1688Binding{},
		&exampleModel.ExaFileUploadAndDownload{},
	); err != nil {
		t.Fatalf("migrate collector 1688 tables: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
	global.GVA_CONFIG.System.OssType = "local"
	global.GVA_CONFIG.Local.Path = "/uploads/file"
	global.GVA_CONFIG.Local.StorePath = filepath.Join(t.TempDir(), "uploads", "file")
	setCollector1688ImageHTTPClient(t, &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return fakePNGResponse(req), nil
	})})
}

func seedCollector1688ListingItem(t *testing.T, sku string) amazonModel.ListingItem {
	t.Helper()
	family := amazonModel.ListingFamily{
		FamilyName: "1688 测试商品组",
		Status:     "draft",
	}
	if err := global.GVA_DB.Create(&family).Error; err != nil {
		t.Fatalf("create family: %v", err)
	}
	item := amazonModel.ListingItem{
		FamilyID: family.ID,
		Role:     "standalone",
		SKU:      sku,
		Status:   "draft",
	}
	if err := global.GVA_DB.Create(&item).Error; err != nil {
		t.Fatalf("create listing item: %v", err)
	}
	return item
}

func mustCreateCollector1688Task(t *testing.T, service Collector1688Service, listingItemID uint) Create1688CollectTaskRes {
	t.Helper()
	result, err := service.CreateTask(context.Background(), amazonReq.Create1688CollectTaskReq{
		ListingItemID: listingItemID,
		MainImageURL:  "https://img.alicdn.com/task-main.jpg",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	return result
}

func validCollector1688SKUOffers() []commonModel.JSONMap {
	return []commonModel.JSONMap{{
		"skuKey":        "pink",
		"attributeText": "粉色笑脸捏捏球",
		"attrs": commonModel.JSONMap{
			"颜色": "粉色笑脸捏捏球",
		},
		"price":     4.99,
		"priceText": "¥4.99",
		"stock":     98472,
		"stockText": "库存98472个",
	}}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func setCollector1688ImageHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()
	previous := collector1688ImageHTTPClient
	collector1688ImageHTTPClient = client
	t.Cleanup(func() {
		collector1688ImageHTTPClient = previous
	})
}

func fakePNGResponse(req *http.Request) *http.Response {
	header := make(http.Header)
	header.Set("Content-Type", "image/png")
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(fakePNGBytes())),
		Request:    req,
	}
}

func fakePNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
}

func formatJsonValueForTest(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
