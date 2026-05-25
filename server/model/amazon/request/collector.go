package request

import (
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type CollectedProductImageDTO struct {
	Sort        int    `json:"sort"`
	IsMain      bool   `json:"isMain"`
	OriginalURL string `json:"originalUrl"`
}

type CollectedProductUpsertFromExtensionReq struct {
	SiteCode             string                     `json:"siteCode"`
	MarketplaceID        string                     `json:"marketplaceId"`
	ASIN                 string                     `json:"asin"`
	ParentASIN           string                     `json:"parentAsin"`
	Title                string                     `json:"title"`
	Brand                string                     `json:"brand"`
	ProductURL           string                     `json:"productUrl"`
	PriceAmount          *float64                   `json:"priceAmount"`
	CurrencyCode         string                     `json:"currencyCode"`
	ListPriceAmount      *float64                   `json:"listPriceAmount"`
	DiscountText         string                     `json:"discountText"`
	RatingValue          *float64                   `json:"ratingValue"`
	ReviewCount          *int                       `json:"reviewCount"`
	BSRText              string                     `json:"bsrText"`
	CategoryPath         []string                   `json:"categoryPath"`
	BrowseNodes          []commonModel.JSONMap      `json:"browseNodes"`
	SellerName           string                     `json:"sellerName"`
	FulfillmentChannel   string                     `json:"fulfillmentChannel"`
	DeliveryEstimateText string                     `json:"deliveryEstimateText"`
	BulletPoints         []string                   `json:"bulletPoints"`
	DescriptionText      string                     `json:"descriptionText"`
	AplusHTML            string                     `json:"aplusHtml"`
	SpecAttributes       commonModel.JSONMap        `json:"specAttributes"`
	VariantSummary       commonModel.JSONMap        `json:"variantSummary"`
	MainImageURL         string                     `json:"mainImageUrl"`
	GalleryImageURLs     []string                   `json:"galleryImageUrls"`
	Images               []CollectedProductImageDTO `json:"images"`
	CollectWarnings      []string                   `json:"collectWarnings"`
	RawPayload           commonModel.JSONMap        `json:"rawPayload"`
}

type CollectedProductListReq struct {
	commonReq.PageInfo
	SiteCode        string `json:"siteCode" form:"siteCode"`
	CollectStatus   string `json:"collectStatus" form:"collectStatus"`
	Brand           string `json:"brand" form:"brand"`
	CategoryLeaf    string `json:"categoryLeaf" form:"categoryLeaf"`
	CategoryKeyword string `json:"categoryKeyword" form:"categoryKeyword"`
}

type CollectedProductFindReq struct {
	ID uint `json:"id" form:"id"`
}

type CollectedProductDeleteReq struct {
	ID uint `json:"id"`
}

type CollectedProductRebindImagesReq struct {
	ID uint `json:"id"`
}

type CollectedProductUpdateRiskReq struct {
	ID                           uint   `json:"id"`
	InfringementStatus           string `json:"infringementStatus"`
	InfringementScreenshotFileID *uint  `json:"infringementScreenshotFileId"`
}

type CollectedProductCategoryListReq struct {
	SiteCode string `json:"siteCode" form:"siteCode"`
}

type CollectedProductSyncToListingReq struct {
	ID         uint `json:"id"`
	StoreID    uint `json:"storeId"`
	TemplateID uint `json:"templateId"`
}
