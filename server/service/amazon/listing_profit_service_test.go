package amazon

import (
	"math"
	"testing"

	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
)

func TestCalculateListingProfitProfileFBA(t *testing.T) {
	offerPrice := 10.0
	exchangeRate := 7.2
	referralRate := 15.0
	adRate := 5.0
	procurementCost := 20.0
	firstLegCost := 5.0
	fbaFee := 8.0
	otherCost := 2.0

	profile := calculateListingProfitProfile("USD", &offerPrice, &amazonReq.ListingProfitProfileDTO{
		FulfillmentMode:      "fba",
		ExchangeRateToCNY:    &exchangeRate,
		ReferralFeeRate:      &referralRate,
		AdCostRate:           &adRate,
		ProcurementCostCNY:   &procurementCost,
		FirstLegCostCNY:      &firstLegCost,
		FBAFulfillmentFeeCNY: &fbaFee,
		OtherCostCNY:         &otherCost,
	})

	if profile.ValidationStatus != profitValidationStatusValid {
		t.Fatalf("expected valid status, got %s (%s)", profile.ValidationStatus, profile.ValidationMessage)
	}
	if profile.Result == nil {
		t.Fatalf("expected profit result")
	}

	assertAlmostEqual(t, "saleCNY", profile.Result.SaleCNY, 72)
	assertAlmostEqual(t, "commissionCNY", profile.Result.CommissionCNY, 10.8)
	assertAlmostEqual(t, "adCostCNY", profile.Result.AdCostCNY, 3.6)
	assertAlmostEqual(t, "fixedCostCNY", profile.Result.FixedCostCNY, 35)
	assertAlmostEqual(t, "grossProfitCNY", profile.Result.GrossProfitCNY, 37)
	assertAlmostEqual(t, "netProfitCNY", profile.Result.NetProfitCNY, 22.6)
	assertAlmostEqual(t, "netMarginRate", profile.Result.NetMarginRate, 0.3139)
	assertAlmostEqual(t, "roiRate", profile.Result.ROIRate, 0.4575)
	assertAlmostEqual(t, "breakEvenPrice", profile.Result.BreakEvenPrice, 6.08)
}

func TestCalculateListingProfitProfileFBM(t *testing.T) {
	offerPrice := 20.0
	exchangeRate := 1.3
	referralRate := 15.0
	adRate := 8.0
	procurementCost := 50.0
	firstLegCost := 10.0
	lastMileCost := 15.0
	otherCost := 5.0

	profile := calculateListingProfitProfile("CAD", &offerPrice, &amazonReq.ListingProfitProfileDTO{
		FulfillmentMode:    "fbm",
		ExchangeRateToCNY:  &exchangeRate,
		ReferralFeeRate:    &referralRate,
		AdCostRate:         &adRate,
		ProcurementCostCNY: &procurementCost,
		FirstLegCostCNY:    &firstLegCost,
		FBMLastMileCostCNY: &lastMileCost,
		OtherCostCNY:       &otherCost,
	})

	if profile.ValidationStatus != profitValidationStatusValid {
		t.Fatalf("expected valid status, got %s (%s)", profile.ValidationStatus, profile.ValidationMessage)
	}
	if profile.Result == nil {
		t.Fatalf("expected profit result")
	}

	assertAlmostEqual(t, "saleCNY", profile.Result.SaleCNY, 26)
	assertAlmostEqual(t, "commissionCNY", profile.Result.CommissionCNY, 3.9)
	assertAlmostEqual(t, "adCostCNY", profile.Result.AdCostCNY, 2.08)
	assertAlmostEqual(t, "fixedCostCNY", profile.Result.FixedCostCNY, 80)
	assertAlmostEqual(t, "grossProfitCNY", profile.Result.GrossProfitCNY, -54)
	assertAlmostEqual(t, "netProfitCNY", profile.Result.NetProfitCNY, -59.98)
	assertAlmostEqual(t, "netMarginRate", profile.Result.NetMarginRate, -2.3069)
	assertAlmostEqual(t, "roiRate", profile.Result.ROIRate, -0.6975)
	assertAlmostEqual(t, "breakEvenPrice", profile.Result.BreakEvenPrice, 79.92)
}

func TestCalculateListingProfitProfileRejectsInvalidRates(t *testing.T) {
	offerPrice := 12.0
	exchangeRate := 7.1
	referralRate := 60.0
	adRate := 40.0

	profile := calculateListingProfitProfile("USD", &offerPrice, &amazonReq.ListingProfitProfileDTO{
		FulfillmentMode:   "fba",
		ExchangeRateToCNY: &exchangeRate,
		ReferralFeeRate:   &referralRate,
		AdCostRate:        &adRate,
	})

	if profile.ValidationStatus != profitValidationStatusInvalid {
		t.Fatalf("expected invalid status, got %s", profile.ValidationStatus)
	}
	if profile.ValidationMessage != "平台佣金率与广告占比之和必须小于 100%" {
		t.Fatalf("unexpected validation message: %s", profile.ValidationMessage)
	}
	if profile.Result != nil {
		t.Fatalf("expected nil result for invalid rate combination")
	}
}

func TestCalculateListingProfitProfileRequiresModeSpecificCost(t *testing.T) {
	offerPrice := 12.0
	exchangeRate := 7.1

	fbaProfile := calculateListingProfitProfile("USD", &offerPrice, &amazonReq.ListingProfitProfileDTO{
		FulfillmentMode:   "fba",
		ExchangeRateToCNY: &exchangeRate,
	})
	if fbaProfile.ValidationMessage != "FBA 模式必须填写 FBA 配送费" {
		t.Fatalf("unexpected FBA validation message: %s", fbaProfile.ValidationMessage)
	}

	fbmProfile := calculateListingProfitProfile("USD", &offerPrice, &amazonReq.ListingProfitProfileDTO{
		FulfillmentMode:   "fbm",
		ExchangeRateToCNY: &exchangeRate,
	})
	if fbmProfile.ValidationMessage != "FBM 模式必须填写尾程派送费" {
		t.Fatalf("unexpected FBM validation message: %s", fbmProfile.ValidationMessage)
	}
}

func TestCalculateListingProfitProfileRequiresPositivePriceAndExchangeRate(t *testing.T) {
	exchangeRate := 7.1
	offerPrice := 0.0
	profile := calculateListingProfitProfile("USD", &offerPrice, &amazonReq.ListingProfitProfileDTO{
		FulfillmentMode:      "fba",
		ExchangeRateToCNY:    &exchangeRate,
		FBAFulfillmentFeeCNY: float64Ptr(8),
	})
	if profile.ValidationMessage != "售价必须大于 0" {
		t.Fatalf("unexpected price validation message: %s", profile.ValidationMessage)
	}

	offerPrice = 20
	exchangeRate = 0
	profile = calculateListingProfitProfile("USD", &offerPrice, &amazonReq.ListingProfitProfileDTO{
		FulfillmentMode:      "fba",
		ExchangeRateToCNY:    &exchangeRate,
		FBAFulfillmentFeeCNY: float64Ptr(8),
	})
	if profile.ValidationMessage != "汇率必须大于 0" {
		t.Fatalf("unexpected exchange validation message: %s", profile.ValidationMessage)
	}
}

func TestSelectListingProfitSummaryPrefersRequestedSiteThenFallsBack(t *testing.T) {
	usProfit := 12.5
	usMargin := 0.18
	caProfit := 9.5
	caMargin := 0.08

	siteCode, mode, netProfit, netMargin, status := selectListingProfitSummary([]ListingMarketplaceBinding{
		{
			SiteCode: "US",
			ProfitProfile: &ListingProfitProfile{
				FulfillmentMode: "fba",
				Result: &ListingProfitResult{
					NetProfitCNY:  &usProfit,
					NetMarginRate: &usMargin,
				},
			},
		},
		{
			SiteCode: "CA",
			ProfitProfile: &ListingProfitProfile{
				FulfillmentMode: "fbm",
				Result: &ListingProfitResult{
					NetProfitCNY:  &caProfit,
					NetMarginRate: &caMargin,
				},
			},
		},
	}, "CA")

	if siteCode != "CA" || mode != "fbm" || status != profitStatusWarning {
		t.Fatalf("unexpected preferred summary: %s %s %s", siteCode, mode, status)
	}
	assertAlmostEqual(t, "preferred net profit", netProfit, 9.5)
	assertAlmostEqual(t, "preferred net margin", netMargin, 0.08)

	siteCode, mode, _, _, status = selectListingProfitSummary([]ListingMarketplaceBinding{
		{
			SiteCode: "CA",
			ProfitProfile: &ListingProfitProfile{
				FulfillmentMode: "fbm",
				Result: &ListingProfitResult{
					NetProfitCNY:  &caProfit,
					NetMarginRate: &caMargin,
				},
			},
		},
		{
			SiteCode: "US",
			ProfitProfile: &ListingProfitProfile{
				FulfillmentMode: "fba",
				Result: &ListingProfitResult{
					NetProfitCNY:  &usProfit,
					NetMarginRate: &usMargin,
				},
			},
		},
	}, "")
	if siteCode != "US" || mode != "fba" || status != profitStatusSuccess {
		t.Fatalf("unexpected fallback summary: %s %s %s", siteCode, mode, status)
	}
}

func assertAlmostEqual(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s should not be nil", name)
	}
	if math.Abs(*got-want) > 0.0001 {
		t.Fatalf("%s mismatch: got %.4f want %.4f", name, *got, want)
	}
}
