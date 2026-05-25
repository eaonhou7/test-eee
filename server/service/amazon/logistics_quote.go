package amazon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const volumetricWeightDivisorCM = 8000.0

type LogisticsQuoteService struct{}

func (s *LogisticsQuoteService) QuoteUS(ctx context.Context, req LogisticsQuoteRequest) (LogisticsQuoteResponse, error) {
	if err := validateLogisticsQuoteRequest(req); err != nil {
		return LogisticsQuoteResponse{}, err
	}

	resp := LogisticsQuoteResponse{
		Request: req,
		Sources: LogisticsQuoteSources{
			Yuntu:  LogisticsQuoteSource{Provider: "yuntu", SourceMode: "database"},
			Yanwen: LogisticsQuoteSource{Provider: "yanwen", SourceMode: "database"},
			Santai: LogisticsQuoteSource{Provider: "santai", SourceMode: "database"},
		},
		ProviderErrors: map[string]string{},
		Quotes:         []LogisticsQuote{},
	}

	for _, provider := range []string{"yuntu", "yanwen", "santai"} {
		data, err := logisticsRepositoryApp.getProviderQuoteData(ctx, provider)
		switch provider {
		case "yuntu":
			resp.Sources.Yuntu = data.Source
		case "yanwen":
			resp.Sources.Yanwen = data.Source
		case "santai":
			resp.Sources.Santai = data.Source
		}
		if err != nil {
			resp.ProviderErrors[provider] = err.Error()
			continue
		}

		for _, channel := range data.Channels {
			if strings.TrimSpace(channel.CountryLabel) != "" && strings.TrimSpace(channel.CountryLabel) != "美国" {
				continue
			}
			if !channelMatchesPlatform(channel, req.Platform) {
				continue
			}
			if !channelMatchesBattery(channel, req.ContainsBattery) {
				continue
			}
			quote, err := quoteLogisticsChannel(channel, req, "database")
			if err != nil {
				continue
			}
			resp.Quotes = append(resp.Quotes, quote)
		}
	}

	sort.Slice(resp.Quotes, func(i, j int) bool {
		if resp.Quotes[i].PriceCNY == resp.Quotes[j].PriceCNY {
			if resp.Quotes[i].Provider == resp.Quotes[j].Provider {
				return resp.Quotes[i].ChannelName < resp.Quotes[j].ChannelName
			}
			return resp.Quotes[i].Provider < resp.Quotes[j].Provider
		}
		return resp.Quotes[i].PriceCNY < resp.Quotes[j].PriceCNY
	})

	for _, quote := range resp.Quotes {
		switch quote.Provider {
		case "yuntu":
			if resp.ProviderLowest.Yuntu == nil {
				copyQuote := quote
				resp.ProviderLowest.Yuntu = &copyQuote
			}
		case "yanwen":
			if resp.ProviderLowest.Yanwen == nil {
				copyQuote := quote
				resp.ProviderLowest.Yanwen = &copyQuote
			}
		case "santai":
			if resp.ProviderLowest.Santai == nil {
				copyQuote := quote
				resp.ProviderLowest.Santai = &copyQuote
			}
		}
		if resp.OverallLowest == nil {
			copyQuote := quote
			resp.OverallLowest = &copyQuote
		}
	}

	if len(resp.ProviderErrors) == 0 {
		resp.ProviderErrors = nil
	}

	if resp.OverallLowest == nil {
		if len(resp.ProviderErrors) == 0 {
			return resp, errors.New("no matching quotes found")
		}
		return resp, errors.New("no provider results available")
	}

	return resp, nil
}

func validateLogisticsQuoteRequest(req LogisticsQuoteRequest) error {
	if req.WeightKG <= 0 {
		return errors.New("weight_kg must be greater than 0")
	}
	hasAnyDimension := req.LengthCM != nil || req.WidthCM != nil || req.HeightCM != nil
	if hasAnyDimension && !req.HasDimensions() {
		return errors.New("length_cm, width_cm, height_cm must be provided together")
	}
	if req.HasDimensions() {
		if *req.LengthCM <= 0 || *req.WidthCM <= 0 || *req.HeightCM <= 0 {
			return errors.New("dimensions must be greater than 0")
		}
	}
	return nil
}

func channelMatchesPlatform(channel logisticsChannel, platform string) bool {
	selectedPlatform := normalizeLogisticsPlatformFilter(platform)
	if selectedPlatform == "" || selectedPlatform == logisticsPlatformAll {
		return true
	}

	channelPlatform := displayLogisticsPlatform(channel.Platform, channel.LogisticsProvider, channel.ChannelName, channel.SheetName)
	normalizedChannelPlatform := normalizeLogisticsPlatformFilter(channelPlatform)
	return normalizedChannelPlatform == logisticsPlatformAll || normalizedChannelPlatform == selectedPlatform
}

func channelMatchesBattery(channel logisticsChannel, containsBattery bool) bool {
	if containsBattery {
		return channel.SupportsBattery || channel.RequiresBattery
	}
	return !hasStringTag(channel.Tags, "pure_electric") && !channel.RequiresBattery
}

func quoteLogisticsChannel(channel logisticsChannel, req LogisticsQuoteRequest, sourceMode string) (LogisticsQuote, error) {
	billableWeight, volumetricWeight, notes := computeBillableWeight(channel, req)
	volumeRatio := 0.0
	if req.WeightKG > 0 && volumetricWeight > 0 {
		volumeRatio = volumetricWeight / req.WeightKG
	}
	row, err := findRateRow(channel.Rows, billableWeight, volumeRatio)
	if err != nil {
		return LogisticsQuote{}, err
	}

	if row.MinBillableWeight > 0 && billableWeight < row.MinBillableWeight {
		billableWeight = row.MinBillableWeight
		notes = append(notes, fmt.Sprintf("row_min_billable=%.4fkg", round4(row.MinBillableWeight)))
	}
	chargeWeight := billableWeight
	switch strings.ToLower(strings.TrimSpace(row.BillableWeightMode)) {
	case "actual":
		chargeWeight = req.WeightKG
	case "volumetric":
		if volumetricWeight > 0 {
			chargeWeight = volumetricWeight
		}
	}
	if row.MinBillableWeight > 0 && chargeWeight < row.MinBillableWeight {
		chargeWeight = row.MinBillableWeight
	}
	if channel.StepWeightKG > 0 {
		chargeWeight = roundUpToStep(chargeWeight, channel.StepWeightKG)
	}

	baseCharge := 0.0
	handling := row.HandlingFeeCNY
	registration := row.RegistrationFeeCNY

	switch channel.RateKind {
	case "per_kg":
		baseCharge = chargeWeight * row.RatePerKG
	case "volume_ratio_per_kg":
		baseCharge = chargeWeight * row.RatePerKG
	case "first_continue":
		baseCharge = calculateSteppedPrice(billableWeight, row)
	default:
		return LogisticsQuote{}, fmt.Errorf("unsupported rate kind %q", channel.RateKind)
	}

	mandatoryFee, sizeWarnings := calculateSizeFees(channel, req)
	warnings := append([]string{}, channel.Warnings...)
	warnings = append(warnings, sizeWarnings...)
	if !req.HasDimensions() {
		warnings = append(warnings, "未提供长宽高，体积重与超尺寸附加费未计入，当前价格为按实际重量估算")
	}
	if channel.ZoneBased {
		warnings = append(warnings, "该渠道按美国分区或邮编计价，当前按美国可用分区最低价估算")
	}
	if channel.RateKind == "volume_ratio_per_kg" && !req.HasDimensions() {
		warnings = append(warnings, "该渠道按体积比档位计价，未提供长宽高时按体积比≤1与实重估算")
	}

	feeBreakdown := LogisticsFeeBreakdown{
		BaseChargeCNY:      round2(baseCharge),
		HandlingFeeCNY:     round2(handling),
		RegistrationFeeCNY: round2(registration),
		MandatoryFeeCNY:    round2(mandatoryFee),
		UnresolvedFees:     uniqueStrings(channel.UnresolvedFees),
		CalculationNotes:   uniqueStrings(notes),
	}
	total := feeBreakdown.BaseChargeCNY + feeBreakdown.HandlingFeeCNY + feeBreakdown.RegistrationFeeCNY + feeBreakdown.MandatoryFeeCNY
	status := "calculated"
	if !req.HasDimensions() {
		status = "estimated"
	}

	var volumetricPtr *float64
	if volumetricWeight > 0 {
		value := round4(volumetricWeight)
		volumetricPtr = &value
	}

	return LogisticsQuote{
		ChannelVersionID:   channel.ChannelVersionID,
		Provider:           channel.Provider,
		LogisticsProvider:  channel.LogisticsProvider,
		Platform:           displayLogisticsPlatform(channel.Platform, channel.LogisticsProvider, channel.ChannelName, channel.SheetName),
		ChannelName:        channel.ChannelName,
		SheetName:          channel.SheetName,
		ServiceCode:        channel.ServiceCode,
		TransitTime:        compactTransitTime(defaultString(channel.TransitTime, row.TransitTime)),
		PriceCNY:           round2(total),
		PriceStatus:        status,
		ActualWeightKG:     round4(req.WeightKG),
		BillableWeightKG:   round4(billableWeight),
		VolumetricWeightKG: volumetricPtr,
		FeeBreakdown:       feeBreakdown,
		ChannelTags:        uniqueStrings(channel.Tags),
		Warnings:           uniqueStrings(warnings),
		SourceMode:         sourceMode,
	}, nil
}

func computeBillableWeight(channel logisticsChannel, req LogisticsQuoteRequest) (float64, float64, []string) {
	actual := req.WeightKG
	billable := actual
	volumetric := 0.0
	notes := []string{fmt.Sprintf("actual_weight=%.4fkg", round4(actual))}

	if req.HasDimensions() {
		divisor := volumetricWeightDivisorCM
		if channel.VolumeDivisor > 0 {
			divisor = channel.VolumeDivisor
		}
		volumetric = (*req.LengthCM) * (*req.WidthCM) * (*req.HeightCM) / divisor
		notes = append(notes, fmt.Sprintf("volumetric_weight=%.4fkg", round4(volumetric)))
		if channel.IgnoreVolumetric && channel.VolumeDivisor <= 0 {
			notes = append(notes, "billable_base=actual_weight")
		} else {
			billable = math.Max(actual, volumetric)
			notes = append(notes, fmt.Sprintf("billable_base=max(actual, volumetric)=%.4fkg", round4(billable)))
		}
	}

	minBillable := channel.MinBillableWeightKG
	if minBillable > 0 && billable < minBillable {
		billable = minBillable
		notes = append(notes, fmt.Sprintf("min_billable=%.4fkg", round4(minBillable)))
	}
	if channel.StepWeightKG > 0 {
		billable = roundUpToStep(billable, channel.StepWeightKG)
		notes = append(notes, fmt.Sprintf("rounded_step=%.4fkg", round4(channel.StepWeightKG)))
	}
	return billable, volumetric, notes
}

func findRateRow(rows []logisticsRateRow, billableWeight, volumeRatio float64) (logisticsRateRow, error) {
	for _, row := range rows {
		minWeight := row.WeightMinKG
		if row.MinBillableWeight > minWeight {
			minWeight = row.MinBillableWeight
		}
		if billableWeight+1e-9 < minWeight {
			continue
		}
		if row.WeightMaxKG > 0 && billableWeight-1e-9 > row.WeightMaxKG {
			continue
		}
		if !volumeRatioMatches(row, volumeRatio) {
			continue
		}
		return row, nil
	}
	return logisticsRateRow{}, errors.New("rate row not found")
}

func volumeRatioMatches(row logisticsRateRow, volumeRatio float64) bool {
	if row.VolumeRatioMin == 0 && row.VolumeRatioMax == 0 {
		return true
	}
	if volumeRatio == 0 {
		volumeRatio = 1
	}
	if row.VolumeRatioMin > 0 && volumeRatio <= row.VolumeRatioMin+1e-9 {
		return false
	}
	if row.VolumeRatioMax > 0 && volumeRatio-row.VolumeRatioMax > 1e-9 {
		return false
	}
	return true
}

func calculateSteppedPrice(billableWeight float64, row logisticsRateRow) float64 {
	if row.FirstWeightKG <= 0 && row.ContinueWeightKG > 0 && row.ContinuePriceCNY > 0 {
		return math.Ceil(billableWeight/row.ContinueWeightKG) * row.ContinuePriceCNY
	}
	if row.FirstWeightKG <= 0 {
		return row.FirstPriceCNY
	}
	if billableWeight <= row.FirstWeightKG {
		return row.FirstPriceCNY
	}
	if row.ContinueWeightKG <= 0 || row.ContinuePriceCNY <= 0 {
		return row.FirstPriceCNY
	}
	extraWeight := billableWeight - row.FirstWeightKG
	units := math.Ceil(extraWeight / row.ContinueWeightKG)
	return row.FirstPriceCNY + units*row.ContinuePriceCNY
}

func calculateSizeFees(channel logisticsChannel, req LogisticsQuoteRequest) (float64, []string) {
	if !req.HasDimensions() {
		return 0, nil
	}
	rules := channel.SizeRules
	length := *req.LengthCM
	width := *req.WidthCM
	height := *req.HeightCM
	girth := length + 2*(width+height)
	fees := 0.0
	warnings := []string{}

	if rules.RejectLengthCM > 0 && length >= rules.RejectLengthCM {
		warnings = append(warnings, fmt.Sprintf("长度 %.1fcm 超过渠道拒收限制 %.1fcm", length, rules.RejectLengthCM))
	}
	if rules.RejectGirthCM > 0 && girth > rules.RejectGirthCM {
		warnings = append(warnings, fmt.Sprintf("围长 %.1fcm 超过渠道拒收限制 %.1fcm", girth, rules.RejectGirthCM))
	}
	if rules.OverLengthFeeCNY > 0 && rules.OverLengthThresholdCM > 0 && maxFloat(length, width, height) > rules.OverLengthThresholdCM {
		fees += rules.OverLengthFeeCNY
	}
	if rules.OverVolumeFeeCNY > 0 && ((rules.OverVolumeSideCM > 0 && overVolumeSides(width, height, length, rules.OverVolumeSideCM)) || (rules.OverVolumeGirthCM > 0 && girth > rules.OverVolumeGirthCM)) {
		fees += rules.OverVolumeFeeCNY
	}
	if rules.OversizeFeeCNY > 0 && ((rules.OversizeMaxLengthCM > 0 && maxFloat(length, width, height) > rules.OversizeMaxLengthCM) || (rules.OversizeMaxWidthCM > 0 && secondMax(length, width, height) > rules.OversizeMaxWidthCM) || (rules.OversizeMaxHeightCM > 0 && minDimension(length, width, height) > rules.OversizeMaxHeightCM) || (rules.OversizeMaxGirthCM > 0 && girth > rules.OversizeMaxGirthCM)) {
		fees += rules.OversizeFeeCNY
	}
	if rules.NeedsCartonAboveLength > 0 && length > rules.NeedsCartonAboveLength && rules.NeedsCartonAboveWidth > 0 && width > rules.NeedsCartonAboveWidth && rules.NeedsCartonAboveHeight > 0 && height > rules.NeedsCartonAboveHeight {
		warnings = append(warnings, "该渠道超过指定尺寸后要求纸箱包装，当前仅返回运费估算")
	}
	return round2(fees), warnings
}

func roundUpToStep(value, step float64) float64 {
	if step <= 0 {
		return value
	}
	return math.Ceil(value/step) * step
}

func overVolumeSides(width, height, length, threshold float64) bool {
	sides := []float64{length, width, height}
	count := 0
	for _, side := range sides {
		if side > threshold {
			count++
		}
	}
	return count >= 2
}

func maxFloat(values ...float64) float64 {
	result := 0.0
	for i, value := range values {
		if i == 0 || value > result {
			result = value
		}
	}
	return result
}

func minDimension(values ...float64) float64 {
	result := 0.0
	for i, value := range values {
		if i == 0 || value < result {
			result = value
		}
	}
	return result
}

func secondMax(values ...float64) float64 {
	sorted := append([]float64{}, values...)
	sort.Float64s(sorted)
	if len(sorted) < 2 {
		return 0
	}
	return sorted[len(sorted)-2]
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
