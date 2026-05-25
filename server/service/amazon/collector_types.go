package amazon

import (
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
)

type CollectedProductImageItem struct {
	ID             uint            `json:"id"`
	Sort           int             `json:"sort"`
	IsMain         bool            `json:"isMain"`
	OriginalURL    string          `json:"originalUrl"`
	FileID         *uint           `json:"fileId,omitempty"`
	File           *FileAssetBrief `json:"file,omitempty"`
	MaterialStatus string          `json:"materialStatus"`
	MaterialError  string          `json:"materialError"`
}

type CollectedProductListItem struct {
	ID                    uint     `json:"id"`
	SiteCode              string   `json:"siteCode"`
	MarketplaceID         string   `json:"marketplaceId"`
	ASIN                  string   `json:"asin"`
	ParentASIN            string   `json:"parentAsin"`
	Title                 string   `json:"title"`
	Brand                 string   `json:"brand"`
	ProductURL            string   `json:"productUrl"`
	PriceAmount           *float64 `json:"priceAmount,omitempty"`
	CurrencyCode          string   `json:"currencyCode"`
	RatingValue           *float64 `json:"ratingValue,omitempty"`
	ReviewCount           *int     `json:"reviewCount,omitempty"`
	BSRText               string   `json:"bsrText"`
	CategoryRoot          string   `json:"categoryRoot"`
	CategoryLeaf          string   `json:"categoryLeaf"`
	CategoryPathText      string   `json:"categoryPathText"`
	SellerName            string   `json:"sellerName"`
	FulfillmentChannel    string   `json:"fulfillmentChannel"`
	DeliveryEstimateText  string   `json:"deliveryEstimateText"`
	MainImageFileID       *uint    `json:"mainImageFileId,omitempty"`
	MainImageURL          string   `json:"mainImageUrl"`
	InfringementStatus    string   `json:"infringementStatus"`
	SyncedListingFamilyID *uint    `json:"syncedListingFamilyId,omitempty"`
	ImageCount            int      `json:"imageCount"`
	CollectStatus         string   `json:"collectStatus"`
	CollectWarnings       []string `json:"collectWarnings"`
	CollectedAt           string   `json:"collectedAt,omitempty"`
	LastCollectedAt       string   `json:"lastCollectedAt,omitempty"`
}

type CollectedProductPageResult struct {
	List     []CollectedProductListItem `json:"list"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}

type CollectedProductDetailRes struct {
	ID                           uint                        `json:"id"`
	SiteCode                     string                      `json:"siteCode"`
	MarketplaceID                string                      `json:"marketplaceId"`
	ASIN                         string                      `json:"asin"`
	ParentASIN                   string                      `json:"parentAsin"`
	Title                        string                      `json:"title"`
	Brand                        string                      `json:"brand"`
	ProductURL                   string                      `json:"productUrl"`
	PriceAmount                  *float64                    `json:"priceAmount,omitempty"`
	CurrencyCode                 string                      `json:"currencyCode"`
	ListPriceAmount              *float64                    `json:"listPriceAmount,omitempty"`
	DiscountText                 string                      `json:"discountText"`
	RatingValue                  *float64                    `json:"ratingValue,omitempty"`
	ReviewCount                  *int                        `json:"reviewCount,omitempty"`
	BSRText                      string                      `json:"bsrText"`
	CategoryPath                 []string                    `json:"categoryPath"`
	CategoryRoot                 string                      `json:"categoryRoot"`
	CategoryLeaf                 string                      `json:"categoryLeaf"`
	CategoryPathText             string                      `json:"categoryPathText"`
	BrowseNodes                  []commonModel.JSONMap       `json:"browseNodes"`
	SellerName                   string                      `json:"sellerName"`
	FulfillmentChannel           string                      `json:"fulfillmentChannel"`
	DeliveryEstimateText         string                      `json:"deliveryEstimateText"`
	BulletPoints                 []string                    `json:"bulletPoints"`
	DescriptionText              string                      `json:"descriptionText"`
	AplusHTML                    string                      `json:"aplusHtml"`
	SpecAttributes               commonModel.JSONMap         `json:"specAttributes"`
	VariantSummary               commonModel.JSONMap         `json:"variantSummary"`
	MainImageFileID              *uint                       `json:"mainImageFileId,omitempty"`
	MainImageURL                 string                      `json:"mainImageUrl"`
	InfringementStatus           string                      `json:"infringementStatus"`
	InfringementScreenshotFileID *uint                       `json:"infringementScreenshotFileId,omitempty"`
	InfringementScreenshot       *FileAssetBrief             `json:"infringementScreenshot,omitempty"`
	SyncedListingFamilyID        *uint                       `json:"syncedListingFamilyId,omitempty"`
	SyncedAt                     string                      `json:"syncedAt,omitempty"`
	ImageCount                   int                         `json:"imageCount"`
	CollectStatus                string                      `json:"collectStatus"`
	CollectWarnings              []string                    `json:"collectWarnings"`
	CollectedAt                  string                      `json:"collectedAt,omitempty"`
	LastCollectedAt              string                      `json:"lastCollectedAt,omitempty"`
	Images                       []CollectedProductImageItem `json:"images"`
	RawPayload                   commonModel.JSONMap         `json:"rawPayload"`
}

type CollectedProductDeleteResult struct {
	ID uint `json:"id"`
}

type CollectedProductCategoryOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type CollectedProductSyncResult struct {
	CollectedProductID uint `json:"collectedProductId"`
	FamilyID           uint `json:"familyId"`
}
