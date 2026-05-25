package amazon

import (
	"context"
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

const (
	profitValidationStatusUnconfigured = "unconfigured"
	profitValidationStatusValid        = "valid"
	profitValidationStatusInvalid      = "invalid"

	profitStatusNone    = "none"
	profitStatusSuccess = "success"
	profitStatusWarning = "warning"
	profitStatusDanger  = "danger"
)

type ProfitService struct{}

func (s *ProfitService) Calculate(_ context.Context, req amazonReq.ListingProfitCalculateReq) (ListingProfitProfile, error) {
	return calculateListingProfitProfile(normalizeCurrencyCode(req.CurrencyCode), req.OfferPrice, req.ProfitProfile), nil
}

func hasListingProfitProfileConfig(profile *amazonReq.ListingProfitProfileDTO) bool {
	return profile != nil && strings.TrimSpace(profile.FulfillmentMode) != ""
}

func syncListingProfitProfile(tx *gorm.DB, binding amazonModel.ListingItemMarketplace, profile *amazonReq.ListingProfitProfileDTO) error {
	var existing amazonModel.ListingProfitProfile
	err := tx.Unscoped().Where("item_marketplace_id = ?", binding.ID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if !hasListingProfitProfileConfig(profile) {
		if err == nil {
			return tx.Unscoped().Delete(&existing).Error
		}
		return nil
	}

	model := buildListingProfitModel(binding.ID, normalizeCurrencyCode(binding.CurrencyCode), binding.OfferPrice, profile)
	if err == nil && !existing.DeletedAt.Valid {
		model.ID = existing.ID
		return tx.Save(&model).Error
	}
	if err == nil {
		if deleteErr := tx.Unscoped().Delete(&existing).Error; deleteErr != nil {
			return deleteErr
		}
	}
	return tx.Create(&model).Error
}

func loadProfitProfilesByMarketplace(ctx context.Context, marketplaceIDs []uint) (map[uint]*ListingProfitProfile, error) {
	result := make(map[uint]*ListingProfitProfile)
	if len(marketplaceIDs) == 0 {
		return result, nil
	}

	var profiles []amazonModel.ListingProfitProfile
	if err := global.GVA_DB.WithContext(ctx).
		Where("item_marketplace_id IN ?", marketplaceIDs).
		Find(&profiles).Error; err != nil {
		return result, err
	}

	for _, profile := range profiles {
		built := buildListingProfitProfileFromModel(profile)
		result[profile.ItemMarketplaceID] = &built
	}
	return result, nil
}

func buildListingProfitModel(itemMarketplaceID uint, currencyCode string, offerPrice *float64, profile *amazonReq.ListingProfitProfileDTO) amazonModel.ListingProfitProfile {
	calculated := calculateListingProfitProfile(currencyCode, offerPrice, profile)
	model := amazonModel.ListingProfitProfile{
		ItemMarketplaceID:     itemMarketplaceID,
		FulfillmentMode:       calculated.FulfillmentMode,
		CostCurrencyCode:      defaultString(strings.TrimSpace(calculated.CostCurrencyCode), "CNY"),
		ExchangeRateToCNY:     cloneFloat64(calculated.ExchangeRateToCNY),
		ReferralFeeRate:       cloneFloat64(calculated.ReferralFeeRate),
		AdCostRate:            cloneFloat64(calculated.AdCostRate),
		ProcurementCostCNY:    cloneFloat64(calculated.ProcurementCostCNY),
		FirstLegCostCNY:       cloneFloat64(calculated.FirstLegCostCNY),
		FBAFulfillmentFeeCNY:  cloneFloat64(calculated.FBAFulfillmentFeeCNY),
		FBMLastMileCostCNY:    cloneFloat64(calculated.FBMLastMileCostCNY),
		OtherCostCNY:          cloneFloat64(calculated.OtherCostCNY),
		ValidationStatus:      calculated.ValidationStatus,
		ValidationMessage:     calculated.ValidationMessage,
		RevenueCurrencyCode:   normalizeCurrencyCode(currencyCode),
		BreakEvenCurrencyCode: normalizeCurrencyCode(currencyCode),
	}
	if calculated.Result != nil {
		model.RevenuePrice = cloneFloat64(calculated.Result.RevenuePrice)
		model.GrossProfitCNY = cloneFloat64(calculated.Result.GrossProfitCNY)
		model.NetProfitCNY = cloneFloat64(calculated.Result.NetProfitCNY)
		model.NetMarginRate = cloneFloat64(calculated.Result.NetMarginRate)
		model.ROIRate = cloneFloat64(calculated.Result.ROIRate)
		model.BreakEvenPrice = cloneFloat64(calculated.Result.BreakEvenPrice)
		model.RevenueCurrencyCode = normalizeCurrencyCode(calculated.Result.RevenueCurrencyCode)
		model.BreakEvenCurrencyCode = normalizeCurrencyCode(calculated.Result.BreakEvenCurrencyCode)
	} else {
		model.RevenuePrice = cloneFloat64(offerPrice)
	}
	if profile != nil && profile.ID > 0 {
		model.ID = profile.ID
	}
	return model
}

func buildListingProfitProfileFromModel(model amazonModel.ListingProfitProfile) ListingProfitProfile {
	profile := ListingProfitProfile{
		ID:                   model.ID,
		FulfillmentMode:      strings.ToLower(strings.TrimSpace(model.FulfillmentMode)),
		CostCurrencyCode:     defaultString(strings.TrimSpace(model.CostCurrencyCode), "CNY"),
		ExchangeRateToCNY:    cloneFloat64(model.ExchangeRateToCNY),
		ReferralFeeRate:      cloneFloat64(model.ReferralFeeRate),
		AdCostRate:           cloneFloat64(model.AdCostRate),
		ProcurementCostCNY:   cloneFloat64(model.ProcurementCostCNY),
		FirstLegCostCNY:      cloneFloat64(model.FirstLegCostCNY),
		FBAFulfillmentFeeCNY: cloneFloat64(model.FBAFulfillmentFeeCNY),
		FBMLastMileCostCNY:   cloneFloat64(model.FBMLastMileCostCNY),
		OtherCostCNY:         cloneFloat64(model.OtherCostCNY),
		ValidationStatus:     defaultString(strings.TrimSpace(model.ValidationStatus), profitValidationStatusUnconfigured),
		ValidationMessage:    strings.TrimSpace(model.ValidationMessage),
	}

	if model.RevenuePrice == nil || model.ExchangeRateToCNY == nil {
		return profile
	}

	commission := round2(*model.RevenuePrice * *model.ExchangeRateToCNY * floatOrZero(model.ReferralFeeRate) / 100)
	adCost := round2(*model.RevenuePrice * *model.ExchangeRateToCNY * floatOrZero(model.AdCostRate) / 100)
	fixedCost := round2(floatOrZero(model.ProcurementCostCNY) + floatOrZero(model.FirstLegCostCNY) + floatOrZero(model.OtherCostCNY) + profitModeSpecificCost(profile.FulfillmentMode, model.FBAFulfillmentFeeCNY, model.FBMLastMileCostCNY))
	saleCNY := round2(*model.RevenuePrice * *model.ExchangeRateToCNY)

	profile.Result = &ListingProfitResult{
		RevenuePrice:          cloneFloat64(model.RevenuePrice),
		RevenueCurrencyCode:   normalizeCurrencyCode(model.RevenueCurrencyCode),
		SaleCNY:               float64Ptr(saleCNY),
		CommissionCNY:         float64Ptr(commission),
		AdCostCNY:             float64Ptr(adCost),
		FixedCostCNY:          float64Ptr(fixedCost),
		GrossProfitCNY:        cloneFloat64(model.GrossProfitCNY),
		NetProfitCNY:          cloneFloat64(model.NetProfitCNY),
		NetMarginRate:         cloneFloat64(model.NetMarginRate),
		ROIRate:               cloneFloat64(model.ROIRate),
		BreakEvenPrice:        cloneFloat64(model.BreakEvenPrice),
		BreakEvenCurrencyCode: normalizeCurrencyCode(model.BreakEvenCurrencyCode),
		CostBreakdown: ListingProfitCostBreakdown{
			ProcurementCostCNY:   cloneFloat64(model.ProcurementCostCNY),
			FirstLegCostCNY:      cloneFloat64(model.FirstLegCostCNY),
			FBAFulfillmentFeeCNY: cloneFloat64(model.FBAFulfillmentFeeCNY),
			FBMLastMileCostCNY:   cloneFloat64(model.FBMLastMileCostCNY),
			OtherCostCNY:         cloneFloat64(model.OtherCostCNY),
			CommissionCNY:        float64Ptr(commission),
			AdCostCNY:            float64Ptr(adCost),
			FixedCostCNY:         float64Ptr(fixedCost),
		},
	}

	return profile
}

func calculateListingProfitProfile(currencyCode string, offerPrice *float64, profile *amazonReq.ListingProfitProfileDTO) ListingProfitProfile {
	result := ListingProfitProfile{
		CostCurrencyCode:  "CNY",
		ReferralFeeRate:   float64Ptr(15),
		ValidationStatus:  profitValidationStatusUnconfigured,
		ValidationMessage: "请选择履约模式后再试算",
	}
	if profile != nil {
		result.ID = profile.ID
		result.FulfillmentMode = strings.ToLower(strings.TrimSpace(profile.FulfillmentMode))
		result.CostCurrencyCode = defaultString(strings.ToUpper(strings.TrimSpace(profile.CostCurrencyCode)), "CNY")
		result.ExchangeRateToCNY = cloneFloat64(profile.ExchangeRateToCNY)
		if profile.ReferralFeeRate != nil {
			result.ReferralFeeRate = cloneFloat64(profile.ReferralFeeRate)
		}
		result.AdCostRate = cloneFloat64(profile.AdCostRate)
		result.ProcurementCostCNY = cloneFloat64(profile.ProcurementCostCNY)
		result.FirstLegCostCNY = cloneFloat64(profile.FirstLegCostCNY)
		result.FBAFulfillmentFeeCNY = cloneFloat64(profile.FBAFulfillmentFeeCNY)
		result.FBMLastMileCostCNY = cloneFloat64(profile.FBMLastMileCostCNY)
		result.OtherCostCNY = cloneFloat64(profile.OtherCostCNY)
	}

	if result.FulfillmentMode == "" {
		return result
	}
	if result.FulfillmentMode != "fba" && result.FulfillmentMode != "fbm" {
		return invalidateListingProfit(result, "履约模式仅支持 FBA 或 FBM")
	}
	if offerPrice == nil || *offerPrice <= 0 {
		return invalidateListingProfit(result, "售价必须大于 0")
	}
	if result.ExchangeRateToCNY == nil || *result.ExchangeRateToCNY <= 0 {
		return invalidateListingProfit(result, "汇率必须大于 0")
	}
	if result.ReferralFeeRate == nil {
		result.ReferralFeeRate = float64Ptr(15)
	}
	if result.AdCostRate == nil {
		result.AdCostRate = float64Ptr(0)
	}
	if hasNegativeFloat(result.ExchangeRateToCNY, result.ReferralFeeRate, result.AdCostRate, result.ProcurementCostCNY, result.FirstLegCostCNY, result.FBAFulfillmentFeeCNY, result.FBMLastMileCostCNY, result.OtherCostCNY) {
		return invalidateListingProfit(result, "利润试算的费率和成本不能为负数")
	}
	if floatOrZero(result.ReferralFeeRate)+floatOrZero(result.AdCostRate) >= 100 {
		return invalidateListingProfit(result, "平台佣金率与广告占比之和必须小于 100%")
	}
	if result.FulfillmentMode == "fba" && result.FBAFulfillmentFeeCNY == nil {
		return invalidateListingProfit(result, "FBA 模式必须填写 FBA 配送费")
	}
	if result.FulfillmentMode == "fbm" && result.FBMLastMileCostCNY == nil {
		return invalidateListingProfit(result, "FBM 模式必须填写尾程派送费")
	}

	saleCNY := round2(*offerPrice * *result.ExchangeRateToCNY)
	commission := round2(saleCNY * floatOrZero(result.ReferralFeeRate) / 100)
	adCost := round2(saleCNY * floatOrZero(result.AdCostRate) / 100)
	fixedCost := round2(floatOrZero(result.ProcurementCostCNY) + floatOrZero(result.FirstLegCostCNY) + floatOrZero(result.OtherCostCNY) + profitModeSpecificCost(result.FulfillmentMode, result.FBAFulfillmentFeeCNY, result.FBMLastMileCostCNY))
	grossProfit := round2(saleCNY - fixedCost)
	netProfit := round2(saleCNY - fixedCost - commission - adCost)
	netMargin := round4(netProfit / saleCNY)

	roiRate := 0.0
	roiDenominator := fixedCost + commission + adCost
	if roiDenominator > 0 {
		roiRate = round4(netProfit / roiDenominator)
	}

	breakEvenBase := (1 - floatOrZero(result.ReferralFeeRate)/100 - floatOrZero(result.AdCostRate)/100) * *result.ExchangeRateToCNY
	if breakEvenBase <= 0 {
		return invalidateListingProfit(result, "当前费率组合无法计算保本售价")
	}
	breakEvenPrice := round2(fixedCost / breakEvenBase)

	result.ValidationStatus = profitValidationStatusValid
	result.ValidationMessage = ""
	result.Result = &ListingProfitResult{
		RevenuePrice:          cloneFloat64(offerPrice),
		RevenueCurrencyCode:   normalizeCurrencyCode(currencyCode),
		SaleCNY:               float64Ptr(saleCNY),
		CommissionCNY:         float64Ptr(commission),
		AdCostCNY:             float64Ptr(adCost),
		FixedCostCNY:          float64Ptr(fixedCost),
		GrossProfitCNY:        float64Ptr(grossProfit),
		NetProfitCNY:          float64Ptr(netProfit),
		NetMarginRate:         float64Ptr(netMargin),
		ROIRate:               float64Ptr(roiRate),
		BreakEvenPrice:        float64Ptr(breakEvenPrice),
		BreakEvenCurrencyCode: normalizeCurrencyCode(currencyCode),
		CostBreakdown: ListingProfitCostBreakdown{
			ProcurementCostCNY:   cloneFloat64(result.ProcurementCostCNY),
			FirstLegCostCNY:      cloneFloat64(result.FirstLegCostCNY),
			FBAFulfillmentFeeCNY: cloneFloat64(result.FBAFulfillmentFeeCNY),
			FBMLastMileCostCNY:   cloneFloat64(result.FBMLastMileCostCNY),
			OtherCostCNY:         cloneFloat64(result.OtherCostCNY),
			CommissionCNY:        float64Ptr(commission),
			AdCostCNY:            float64Ptr(adCost),
			FixedCostCNY:         float64Ptr(fixedCost),
		},
	}

	return result
}

func invalidateListingProfit(profile ListingProfitProfile, message string) ListingProfitProfile {
	profile.ValidationStatus = profitValidationStatusInvalid
	profile.ValidationMessage = message
	profile.Result = nil
	return profile
}

func hasNegativeFloat(values ...*float64) bool {
	for _, value := range values {
		if value != nil && *value < 0 {
			return true
		}
	}
	return false
}

func profitModeSpecificCost(mode string, fbaFee, fbmFee *float64) float64 {
	if mode == "fba" {
		return floatOrZero(fbaFee)
	}
	if mode == "fbm" {
		return floatOrZero(fbmFee)
	}
	return 0
}

func floatOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func normalizeCurrencyCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "USD"
	}
	return value
}

func selectListingProfitSummary(marketplaces []ListingMarketplaceBinding, preferredSiteCode string) (string, string, *float64, *float64, string) {
	candidates := marketplaces
	if len(candidates) == 0 {
		return "", "", nil, nil, profitStatusNone
	}

	pick := func(siteCode string) *ListingMarketplaceBinding {
		for index := range candidates {
			candidate := &candidates[index]
			if strings.ToUpper(strings.TrimSpace(candidate.SiteCode)) != siteCode {
				continue
			}
			if candidate.ProfitProfile == nil || candidate.ProfitProfile.Result == nil || candidate.ProfitProfile.Result.NetProfitCNY == nil || candidate.ProfitProfile.Result.NetMarginRate == nil {
				continue
			}
			return candidate
		}
		return nil
	}

	if normalized := strings.ToUpper(strings.TrimSpace(preferredSiteCode)); normalized != "" {
		if selected := pick(normalized); selected != nil {
			return profitSummaryFromMarketplace(*selected)
		}
	}
	for _, siteCode := range []string{"US", "CA", "MX"} {
		if selected := pick(siteCode); selected != nil {
			return profitSummaryFromMarketplace(*selected)
		}
	}
	for _, marketplace := range candidates {
		if marketplace.ProfitProfile == nil || marketplace.ProfitProfile.Result == nil || marketplace.ProfitProfile.Result.NetProfitCNY == nil || marketplace.ProfitProfile.Result.NetMarginRate == nil {
			continue
		}
		return profitSummaryFromMarketplace(marketplace)
	}
	return "", "", nil, nil, profitStatusNone
}

func profitSummaryFromMarketplace(marketplace ListingMarketplaceBinding) (string, string, *float64, *float64, string) {
	mode := ""
	if marketplace.ProfitProfile != nil {
		mode = marketplace.ProfitProfile.FulfillmentMode
	}
	netProfit := cloneFloat64(marketplace.ProfitProfile.Result.NetProfitCNY)
	netMargin := cloneFloat64(marketplace.ProfitProfile.Result.NetMarginRate)
	return marketplace.SiteCode, mode, netProfit, netMargin, buildListingProfitStatus(netMargin)
}

func buildListingProfitStatus(netMargin *float64) string {
	if netMargin == nil {
		return profitStatusNone
	}
	if *netMargin < 0 {
		return profitStatusDanger
	}
	if *netMargin < 0.1 {
		return profitStatusWarning
	}
	return profitStatusSuccess
}

func fillTreeProfitFallback(node *ListingTreeItem) {
	if node == nil {
		return
	}
	for index := range node.Children {
		fillTreeProfitFallback(&node.Children[index])
	}
	if node.ProfitNetProfitCNY != nil && node.ProfitNetMarginRate != nil {
		return
	}
	for _, child := range node.Children {
		if child.ProfitNetProfitCNY == nil || child.ProfitNetMarginRate == nil {
			continue
		}
		node.ProfitSummarySiteCode = child.ProfitSummarySiteCode
		node.ProfitSummaryMode = child.ProfitSummaryMode
		node.ProfitNetProfitCNY = cloneFloat64(child.ProfitNetProfitCNY)
		node.ProfitNetMarginRate = cloneFloat64(child.ProfitNetMarginRate)
		node.ProfitStatus = child.ProfitStatus
		return
	}
}

func formatProfitModeLabel(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "fba":
		return "FBA"
	case "fbm":
		return "FBM"
	default:
		return strings.ToUpper(strings.TrimSpace(mode))
	}
}

func buildListingProfitSummaryText(siteCode, mode string, netProfit, netMargin *float64) string {
	if netProfit == nil || netMargin == nil {
		return ""
	}
	return fmt.Sprintf("%s %s %.2f / %.2f%%", strings.ToUpper(strings.TrimSpace(siteCode)), formatProfitModeLabel(mode), *netProfit, *netMargin*100)
}
