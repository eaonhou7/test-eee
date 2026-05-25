package amazon

import (
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
)

type Create1688CollectTaskRes struct {
	TaskID             uint   `json:"taskId"`
	TaskToken          string `json:"taskToken"`
	TaskType           string `json:"taskType,omitempty"`
	SearchURL          string `json:"searchUrl"`
	DetailURL          string `json:"detailUrl,omitempty"`
	ExpiresAt          string `json:"expiresAt"`
	SystemCode         string `json:"systemCode"`
	MainImageURL       string `json:"mainImageUrl,omitempty"`
	OfferID            string `json:"offerId,omitempty"`
	CollectedProductID uint   `json:"collectedProductId,omitempty"`
}

type Collected1688TaskResult struct {
	TaskID             uint   `json:"taskId"`
	TaskToken          string `json:"taskToken"`
	Status             string `json:"status"`
	SystemCode         string `json:"systemCode"`
	MainImageURL       string `json:"mainImageUrl"`
	ListingItemID      uint   `json:"listingItemId"`
	ListingFamilyID    uint   `json:"listingFamilyId"`
	CollectedProductID *uint  `json:"collectedProductId,omitempty"`
	OfferID            string `json:"offerId,omitempty"`
	ExpiresAt          string `json:"expiresAt,omitempty"`
}

type Collected1688BindingBrief struct {
	ID                 uint                `json:"id"`
	ListingItemID      uint                `json:"listingItemId"`
	ListingFamilyID    uint                `json:"listingFamilyId"`
	SystemCode         string              `json:"systemCode"`
	CollectedProductID uint                `json:"collectedProductId"`
	TaskID             uint                `json:"taskId"`
	SelectedSKUKey     string              `json:"selectedSkuKey"`
	SelectedSKUAttrs   commonModel.JSONMap `json:"selectedSkuAttrs"`
	MappingStatus      string              `json:"mappingStatus"`
	IsActive           bool                `json:"isActive"`
	BoundAt            string              `json:"boundAt,omitempty"`
	LastCollectedAt    string              `json:"lastCollectedAt,omitempty"`
}

type Collected1688ProductImageItem struct {
	ID             uint            `json:"id"`
	ImageType      string          `json:"imageType"`
	Sort           int             `json:"sort"`
	IsMain         bool            `json:"isMain"`
	OriginalURL    string          `json:"originalUrl"`
	FileID         *uint           `json:"fileId,omitempty"`
	File           *FileAssetBrief `json:"file,omitempty"`
	MaterialStatus string          `json:"materialStatus"`
	MaterialError  string          `json:"materialError"`
}

type Collected1688ProductListItem struct {
	ID               uint                        `json:"id"`
	OfferID          string                      `json:"offerId"`
	Title            string                      `json:"title"`
	ProductURL       string                      `json:"productUrl"`
	SellerCompany    string                      `json:"sellerCompany"`
	ShopName         string                      `json:"shopName"`
	PriceText        string                      `json:"priceText"`
	PriceMin         *float64                    `json:"priceMin,omitempty"`
	PriceMax         *float64                    `json:"priceMax,omitempty"`
	CurrencyCode     string                      `json:"currencyCode"`
	MinOrderQuantity *float64                    `json:"minOrderQuantity,omitempty"`
	OrderUnit        string                      `json:"orderUnit"`
	CategoryPathText string                      `json:"categoryPathText"`
	MainImageFileID  *uint                       `json:"mainImageFileId,omitempty"`
	MainImageURL     string                      `json:"mainImageUrl"`
	ImageCount       int                         `json:"imageCount"`
	CollectStatus    string                      `json:"collectStatus"`
	CollectWarnings  []string                    `json:"collectWarnings"`
	SystemCodeText   string                      `json:"systemCodeText"`
	Bindings         []Collected1688BindingBrief `json:"bindings"`
	CollectedAt      string                      `json:"collectedAt,omitempty"`
	LastCollectedAt  string                      `json:"lastCollectedAt,omitempty"`
}

type Collected1688ProductPageResult struct {
	List     []Collected1688ProductListItem `json:"list"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"pageSize"`
}

type Collected1688ProductDetail struct {
	ID                uint                            `json:"id"`
	OfferID           string                          `json:"offerId"`
	Title             string                          `json:"title"`
	ProductURL        string                          `json:"productUrl"`
	SellerCompany     string                          `json:"sellerCompany"`
	ShopName          string                          `json:"shopName"`
	SellerURL         string                          `json:"sellerUrl"`
	ShopURL           string                          `json:"shopUrl"`
	PriceText         string                          `json:"priceText"`
	PriceMin          *float64                        `json:"priceMin,omitempty"`
	PriceMax          *float64                        `json:"priceMax,omitempty"`
	CurrencyCode      string                          `json:"currencyCode"`
	MinOrderQuantity  *float64                        `json:"minOrderQuantity,omitempty"`
	OrderUnit         string                          `json:"orderUnit"`
	Origin            string                          `json:"origin"`
	FreightText       string                          `json:"freightText"`
	CategoryPath      []string                        `json:"categoryPath"`
	CategoryPathText  string                          `json:"categoryPathText"`
	SpecAttributes    commonModel.JSONMap             `json:"specAttributes"`
	ProductAttributes commonModel.JSONMap             `json:"productAttributes"`
	PackageInfo       commonModel.JSONMap             `json:"packageInfo"`
	SKUAttributes     []commonModel.JSONMap           `json:"skuAttributes"`
	SKUOffers         []commonModel.JSONMap           `json:"skuOffers"`
	DetailSections    []commonModel.JSONMap           `json:"detailSections"`
	DetailText        string                          `json:"detailText"`
	DescriptionHTML   string                          `json:"descriptionHtml"`
	MainImageFileID   *uint                           `json:"mainImageFileId,omitempty"`
	MainImageURL      string                          `json:"mainImageUrl"`
	ImageCount        int                             `json:"imageCount"`
	CollectStatus     string                          `json:"collectStatus"`
	CollectWarnings   []string                        `json:"collectWarnings"`
	CollectedAt       string                          `json:"collectedAt,omitempty"`
	LastCollectedAt   string                          `json:"lastCollectedAt,omitempty"`
	Images            []Collected1688ProductImageItem `json:"images"`
	Bindings          []Collected1688BindingBrief     `json:"bindings"`
	RawPayload        commonModel.JSONMap             `json:"rawPayload"`
}

type Collected1688ProductDeleteResult struct {
	ID uint `json:"id"`
}

type Collected1688UpsertResult struct {
	TaskID             uint   `json:"taskId"`
	CollectedProductID uint   `json:"collectedProductId"`
	ListingItemID      uint   `json:"listingItemId"`
	SystemCode         string `json:"systemCode"`
	OfferID            string `json:"offerId"`
}
