package request

import (
	commonModel "github.com/flipped-aurora/gin-vue-admin/server/model/common"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type Create1688CollectTaskReq struct {
	ListingItemID uint   `json:"listingItemId"`
	SystemCode    string `json:"systemCode"`
	MainImageURL  string `json:"mainImageUrl"`
}

type Create1688RepairTaskReq struct {
	CollectedProductID uint   `json:"collectedProductId"`
	OfferID            string `json:"offerId"`
}

type Report1688CollectTaskStateReq struct {
	TaskToken    string `json:"taskToken"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
}

type Collected1688ProductUpsertFromExtensionReq struct {
	TaskToken         string                `json:"taskToken"`
	OfferID           string                `json:"offerId"`
	Title             string                `json:"title"`
	ProductURL        string                `json:"productUrl"`
	MainImageURL      string                `json:"mainImageUrl"`
	GalleryImageURLs  []string              `json:"galleryImageUrls"`
	DetailImageURLs   []string              `json:"detailImageUrls"`
	PriceText         string                `json:"priceText"`
	PriceMin          *float64              `json:"priceMin"`
	PriceMax          *float64              `json:"priceMax"`
	CurrencyCode      string                `json:"currencyCode"`
	MinOrderQuantity  *float64              `json:"minOrderQuantity"`
	OrderUnit         string                `json:"orderUnit"`
	SellerCompany     string                `json:"sellerCompany"`
	ShopName          string                `json:"shopName"`
	SellerURL         string                `json:"sellerUrl"`
	ShopURL           string                `json:"shopUrl"`
	Origin            string                `json:"origin"`
	FreightText       string                `json:"freightText"`
	CategoryPath      []string              `json:"categoryPath"`
	SpecAttributes    commonModel.JSONMap   `json:"specAttributes"`
	ProductAttributes commonModel.JSONMap   `json:"productAttributes"`
	PackageInfo       commonModel.JSONMap   `json:"packageInfo"`
	SKUAttributes     []commonModel.JSONMap `json:"skuAttributes"`
	SKUOffers         []commonModel.JSONMap `json:"skuOffers"`
	DetailSections    []commonModel.JSONMap `json:"detailSections"`
	DetailText        string                `json:"detailText"`
	DescriptionHTML   string                `json:"descriptionHtml"`
	CollectWarnings   []string              `json:"collectWarnings"`
	RawPayload        commonModel.JSONMap   `json:"rawPayload"`
}

type Collected1688ProductListReq struct {
	commonReq.PageInfo
	CollectStatus string `json:"collectStatus" form:"collectStatus"`
	BindingStatus string `json:"bindingStatus" form:"bindingStatus"`
	SystemCode    string `json:"systemCode" form:"systemCode"`
	ShopKeyword   string `json:"shopKeyword" form:"shopKeyword"`
}

type Collected1688ProductFindReq struct {
	ID uint `json:"id" form:"id"`
}

type Collected1688ProductDeleteReq struct {
	ID uint `json:"id"`
}

type Collect1688BindingVariantMappingReq struct {
	BindingID        uint                `json:"bindingId"`
	SelectedSKUKey   string              `json:"selectedSkuKey"`
	SelectedSKUAttrs commonModel.JSONMap `json:"selectedSkuAttrs"`
}
