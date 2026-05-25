package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type logisticsRepository struct{}

type providerQuoteData struct {
	Source   LogisticsQuoteSource
	Channels []logisticsChannel
}

func (r *logisticsRepository) getProviderQuoteData(ctx context.Context, provider string) (providerQuoteData, error) {
	source, err := r.getProviderSourceSummary(ctx, provider)
	if err != nil {
		return providerQuoteData{}, err
	}

	db := global.GVA_DB.WithContext(ctx)
	var versions []amazonModel.LogisticsChannelVersion
	if err := db.Where("provider = ? AND is_active = ?", provider, true).
		Order("channel_name ASC, id ASC").
		Find(&versions).Error; err != nil {
		return providerQuoteData{}, err
	}
	if len(versions) == 0 {
		return providerQuoteData{Source: source}, fmt.Errorf("%s 没有当前激活报价数据", provider)
	}

	rowsByVersion, err := r.getRateRowsByVersionIDs(ctx, collectChannelVersionIDs(versions))
	if err != nil {
		return providerQuoteData{}, err
	}

	channels := make([]logisticsChannel, 0, len(versions))
	for _, version := range versions {
		channel, err := buildChannelFromVersion(version, rowsByVersion[version.ID])
		if err != nil {
			return providerQuoteData{}, err
		}
		channels = append(channels, channel)
	}
	return providerQuoteData{
		Source:   source,
		Channels: channels,
	}, nil
}

func (r *logisticsRepository) getProviderSourceSummary(ctx context.Context, provider string) (LogisticsQuoteSource, error) {
	db := global.GVA_DB.WithContext(ctx)

	source := LogisticsQuoteSource{
		Provider:   provider,
		SourceMode: "database",
	}

	if err := db.Model(&amazonModel.LogisticsChannelVersion{}).
		Where("provider = ? AND is_active = ?", provider, true).
		Count(&source.ActiveChannelCount).Error; err != nil {
		return source, err
	}

	if err := db.Model(&amazonModel.LogisticsChannelVersion{}).
		Where("provider = ? AND is_active = ?", provider, true).
		Distinct("batch_id").
		Count(&source.ActiveBatchCount).Error; err != nil {
		return source, err
	}

	var latestBatch amazonModel.LogisticsUploadBatch
	err := db.Where("provider = ? AND status = ?", provider, "success").
		Order("created_at DESC").
		First(&latestBatch).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return source, err
	}
	if err == nil {
		source.LatestBatchID = &latestBatch.ID
		source.LatestFileName = latestBatch.SourceFileName
		source.LatestUploadedAt = &latestBatch.CreatedAt
	}

	return source, nil
}

func (r *logisticsRepository) getChannelPage(ctx context.Context, req amazonReq.LogisticsChannelPageReq) (LogisticsChannelPageResult, error) {
	db := global.GVA_DB.WithContext(ctx)
	tableName := amazonModel.LogisticsChannelVersion{}.TableName()
	batchTable := amazonModel.LogisticsUploadBatch{}.TableName()

	query := db.Model(&amazonModel.LogisticsChannelVersion{}).
		Select(tableName+".id", tableName+".batch_id", tableName+".provider", tableName+".logical_product_key",
			tableName+".product_code", tableName+".product_code_type", tableName+".channel_name",
			tableName+".sheet_name", tableName+".logistics_provider", tableName+".platform", tableName+".service_code",
			tableName+".transit_time", tableName+".country_label", tableName+".effective_at", tableName+".effective_text_raw", tableName+".is_active",
			batchTable+".source_file_name", batchTable+".created_at AS uploaded_at").
		Joins("LEFT JOIN " + batchTable + " ON " + batchTable + ".id = " + tableName + ".batch_id")

	query = applyChannelFilters(query, req, tableName, batchTable)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return LogisticsChannelPageResult{}, err
	}

	var list []LogisticsChannelPageItem
	if err := query.
		Scopes(req.PageInfo.Paginate()).
		Order(tableName + ".is_active DESC").
		Order(clause.OrderByColumn{Column: clause.Column{Table: tableName, Name: "effective_at"}, Desc: true}).
		Order(clause.OrderByColumn{Column: clause.Column{Table: tableName, Name: "updated_at"}, Desc: true}).
		Scan(&list).Error; err != nil {
		return LogisticsChannelPageResult{}, err
	}
	for index := range list {
		list[index].Platform = displayLogisticsPlatform(list[index].Platform, list[index].LogisticsProvider, list[index].ChannelName, list[index].SheetName)
		list[index].TransitTime = compactTransitTime(list[index].TransitTime)
	}

	return LogisticsChannelPageResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (r *logisticsRepository) getChannelDetail(ctx context.Context, channelVersionID uint) (LogisticsChannelDetail, error) {
	db := global.GVA_DB.WithContext(ctx)
	tableName := amazonModel.LogisticsChannelVersion{}.TableName()
	batchTable := amazonModel.LogisticsUploadBatch{}.TableName()

	type channelDetailRow struct {
		amazonModel.LogisticsChannelVersion
		SourceFileName      string    `gorm:"column:source_file_name"`
		FileURL             string    `gorm:"column:file_url"`
		UploadedAt          time.Time `gorm:"column:uploaded_at"`
		UploadedBy          uint      `gorm:"column:uploaded_by"`
		UploadedAuthorityID uint      `gorm:"column:uploaded_authority_id"`
	}

	var row channelDetailRow
	if err := db.Model(&amazonModel.LogisticsChannelVersion{}).
		Select(tableName+".*", batchTable+".source_file_name", batchTable+".file_url",
			batchTable+".created_at AS uploaded_at", batchTable+".uploaded_by", batchTable+".uploaded_authority_id").
		Joins("LEFT JOIN "+batchTable+" ON "+batchTable+".id = "+tableName+".batch_id").
		Where(tableName+".id = ?", channelVersionID).
		First(&row).Error; err != nil {
		return LogisticsChannelDetail{}, err
	}

	tags, err := decodeStringSliceJSON(row.TagsJSON)
	if err != nil {
		return LogisticsChannelDetail{}, err
	}
	warnings, err := decodeStringSliceJSON(row.WarningsJSON)
	if err != nil {
		return LogisticsChannelDetail{}, err
	}
	unresolvedFees, err := decodeStringSliceJSON(row.UnresolvedFeesJSON)
	if err != nil {
		return LogisticsChannelDetail{}, err
	}
	sizeRules, err := decodeSizeRulesJSON(row.SizeRulesJSON)
	if err != nil {
		return LogisticsChannelDetail{}, err
	}

	return LogisticsChannelDetail{
		ID:                  row.ID,
		BatchID:             row.BatchID,
		Provider:            row.Provider,
		LogicalProductKey:   row.LogicalProductKey,
		ProductCode:         row.ProductCode,
		ProductCodeType:     row.ProductCodeType,
		ChannelName:         row.ChannelName,
		SheetName:           row.SheetName,
		LogisticsProvider:   row.LogisticsProvider,
		Platform:            displayLogisticsPlatform(row.Platform, row.LogisticsProvider, row.ChannelName, row.SheetName),
		ServiceCode:         row.ServiceCode,
		EffectiveAt:         row.EffectiveAt,
		EffectiveTextRaw:    row.EffectiveTextRaw,
		TransitTime:         compactTransitTime(row.TransitTime),
		CountryLabel:        row.CountryLabel,
		SupportsBattery:     row.SupportsBattery,
		RequiresBattery:     row.RequiresBattery,
		RateKind:            row.RateKind,
		VolumeDivisor:       row.VolumeDivisor,
		VolumeThreshold:     row.VolumeThreshold,
		VolumeThresholdMax:  row.VolumeThresholdMax,
		IgnoreVolumetric:    row.IgnoreVolumetric,
		MinBillableWeightKG: row.MinBillableWeightKG,
		StepWeightKG:        row.StepWeightKG,
		SizeRules:           sizeRules,
		Tags:                tags,
		Warnings:            warnings,
		UnresolvedFees:      unresolvedFees,
		ZoneBased:           row.ZoneBased,
		IsActive:            row.IsActive,
		ActivatedAt:         row.ActivatedAt,
		DeactivatedAt:       row.DeactivatedAt,
		SupersededByBatchID: row.SupersededByBatchID,
		SourceFileName:      row.SourceFileName,
		FileURL:             row.FileURL,
		UploadedAt:          row.UploadedAt,
		UploadedBy:          row.UploadedBy,
		UploadedAuthorityID: row.UploadedAuthorityID,
	}, nil
}

func (r *logisticsRepository) getRateRowPage(ctx context.Context, channelVersionID uint, page, pageSize int) (LogisticsRateRowPageResult, error) {
	db := global.GVA_DB.WithContext(ctx)
	pageInfo := normalizePageInfo(page, pageSize)

	query := db.Model(&amazonModel.LogisticsRateRowVersion{}).
		Where("channel_version_id = ?", channelVersionID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return LogisticsRateRowPageResult{}, err
	}

	var rows []amazonModel.LogisticsRateRowVersion
	if err := query.
		Scopes(pageInfo.Paginate()).
		Order("sequence_no ASC, id ASC").
		Find(&rows).Error; err != nil {
		return LogisticsRateRowPageResult{}, err
	}

	list := make([]LogisticsRateRowItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, LogisticsRateRowItem{
			ID:                 row.ID,
			ChannelVersionID:   row.ChannelVersionID,
			SequenceNo:         row.SequenceNo,
			Zone:               row.Zone,
			WeightMinKG:        row.WeightMinKG,
			WeightMaxKG:        row.WeightMaxKG,
			RatePerKG:          row.RatePerKG,
			HandlingFeeCNY:     row.HandlingFeeCNY,
			RegistrationFeeCNY: row.RegistrationFeeCNY,
			FirstWeightKG:      row.FirstWeightKG,
			FirstPriceCNY:      row.FirstPriceCNY,
			ContinueWeightKG:   row.ContinueWeightKG,
			ContinuePriceCNY:   row.ContinuePriceCNY,
			MinBillableWeight:  row.MinBillableWeight,
			TransitTime:        compactTransitTime(row.TransitTime),
			VolumeRatioMin:     row.VolumeRatioMin,
			VolumeRatioMax:     row.VolumeRatioMax,
			BillableWeightMode: row.BillableWeightMode,
			RateLabelRaw:       row.RateLabelRaw,
		})
	}

	return LogisticsRateRowPageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, nil
}

func (r *logisticsRepository) getVersionPage(ctx context.Context, provider, logicalProductKey string, page, pageSize int) (LogisticsVersionPageResult, error) {
	db := global.GVA_DB.WithContext(ctx)
	pageInfo := normalizePageInfo(page, pageSize)
	tableName := amazonModel.LogisticsChannelVersion{}.TableName()
	batchTable := amazonModel.LogisticsUploadBatch{}.TableName()

	query := db.Model(&amazonModel.LogisticsChannelVersion{}).
		Select(tableName+".id", tableName+".batch_id", tableName+".provider", tableName+".logical_product_key",
			tableName+".product_code", tableName+".product_code_type", tableName+".channel_name", tableName+".sheet_name",
			tableName+".logistics_provider", tableName+".platform", tableName+".service_code", tableName+".effective_at", tableName+".effective_text_raw",
			tableName+".is_active", tableName+".activated_at", tableName+".deactivated_at",
			batchTable+".source_file_name", batchTable+".file_url", batchTable+".created_at AS uploaded_at").
		Joins("LEFT JOIN "+batchTable+" ON "+batchTable+".id = "+tableName+".batch_id").
		Where(tableName+".provider = ? AND "+tableName+".logical_product_key = ?", provider, logicalProductKey)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return LogisticsVersionPageResult{}, err
	}

	var list []LogisticsVersionItem
	if err := query.
		Scopes(pageInfo.Paginate()).
		Order(tableName + ".is_active DESC").
		Order(tableName + ".created_at DESC").
		Scan(&list).Error; err != nil {
		return LogisticsVersionPageResult{}, err
	}
	for index := range list {
		list[index].Platform = displayLogisticsPlatform(list[index].Platform, list[index].LogisticsProvider, list[index].ChannelName, list[index].SheetName)
	}

	return LogisticsVersionPageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, nil
}

func (r *logisticsRepository) markBatchFailed(ctx context.Context, batchID uint, reason string) error {
	return global.GVA_DB.WithContext(ctx).
		Model(&amazonModel.LogisticsUploadBatch{}).
		Where("id = ?", batchID).
		Updates(map[string]interface{}{
			"status":         "failed",
			"failure_reason": reason,
		}).Error
}

func (r *logisticsRepository) markBatchSuccess(ctx context.Context, batchID uint, parsedChannelCount, parsedRateRowCount, touchedProductCount int) error {
	return global.GVA_DB.WithContext(ctx).
		Model(&amazonModel.LogisticsUploadBatch{}).
		Where("id = ?", batchID).
		Updates(map[string]interface{}{
			"status":                "success",
			"parsed_channel_count":  parsedChannelCount,
			"parsed_rate_row_count": parsedRateRowCount,
			"touched_product_count": touchedProductCount,
			"failure_reason":        "",
		}).Error
}

func (r *logisticsRepository) createBatch(ctx context.Context, batch *amazonModel.LogisticsUploadBatch) error {
	return global.GVA_DB.WithContext(ctx).Create(batch).Error
}

func (r *logisticsRepository) saveWorkbookImport(ctx context.Context, batchID uint, provider string, data logisticsWorkbookData) (int, int, error) {
	db := global.GVA_DB.WithContext(ctx)
	now := time.Now().UTC()

	touchedKeys := map[string]struct{}{}
	newVersions := make([]amazonModel.LogisticsChannelVersion, 0, len(data.Channels))
	versionRows := make([][]amazonModel.LogisticsRateRowVersion, 0, len(data.Channels))
	totalRateRows := 0

	for _, channel := range data.Channels {
		record := buildChannelVersionRecord(batchID, provider, channel, now)
		touchedKeys[record.LogicalProductKey] = struct{}{}
		newVersions = append(newVersions, record)

		rows := make([]amazonModel.LogisticsRateRowVersion, 0, len(channel.Rows))
		for index, rateRow := range channel.Rows {
			rows = append(rows, amazonModel.LogisticsRateRowVersion{
				SequenceNo:         index + 1,
				Zone:               rateRow.Zone,
				WeightMinKG:        rateRow.WeightMinKG,
				WeightMaxKG:        rateRow.WeightMaxKG,
				RatePerKG:          rateRow.RatePerKG,
				HandlingFeeCNY:     rateRow.HandlingFeeCNY,
				RegistrationFeeCNY: rateRow.RegistrationFeeCNY,
				FirstWeightKG:      rateRow.FirstWeightKG,
				FirstPriceCNY:      rateRow.FirstPriceCNY,
				ContinueWeightKG:   rateRow.ContinueWeightKG,
				ContinuePriceCNY:   rateRow.ContinuePriceCNY,
				MinBillableWeight:  rateRow.MinBillableWeight,
				TransitTime:        compactTransitTime(rateRow.TransitTime),
				VolumeRatioMin:     rateRow.VolumeRatioMin,
				VolumeRatioMax:     rateRow.VolumeRatioMax,
				BillableWeightMode: rateRow.BillableWeightMode,
				RateLabelRaw:       rateRow.RateLabelRaw,
			})
		}
		totalRateRows += len(rows)
		versionRows = append(versionRows, rows)
	}

	if len(newVersions) == 0 {
		return 0, 0, errors.New("未解析到美国渠道数据")
	}

	keys := mapKeys(touchedKeys)

	return len(newVersions), totalRateRows, db.Transaction(func(tx *gorm.DB) error {
		var activeVersions []amazonModel.LogisticsChannelVersion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("provider = ? AND logical_product_key IN ? AND is_active = ?", provider, keys, true).
			Find(&activeVersions).Error; err != nil {
			return err
		}

		if len(activeVersions) > 0 {
			activeIDs := collectChannelVersionIDs(activeVersions)
			if err := tx.Model(&amazonModel.LogisticsChannelVersion{}).
				Where("id IN ?", activeIDs).
				Updates(map[string]interface{}{
					"is_active":              false,
					"deactivated_at":         now,
					"superseded_by_batch_id": batchID,
				}).Error; err != nil {
				return err
			}
		}

		for index := range newVersions {
			if err := tx.Create(&newVersions[index]).Error; err != nil {
				return err
			}
			for rowIndex := range versionRows[index] {
				versionRows[index][rowIndex].ChannelVersionID = newVersions[index].ID
			}
			if len(versionRows[index]) > 0 {
				if err := tx.Create(&versionRows[index]).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *logisticsRepository) getRateRowsByVersionIDs(ctx context.Context, ids []uint) (map[uint][]amazonModel.LogisticsRateRowVersion, error) {
	result := map[uint][]amazonModel.LogisticsRateRowVersion{}
	if len(ids) == 0 {
		return result, nil
	}

	var rows []amazonModel.LogisticsRateRowVersion
	if err := global.GVA_DB.WithContext(ctx).
		Where("channel_version_id IN ?", ids).
		Order("channel_version_id ASC, sequence_no ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.ChannelVersionID] = append(result[row.ChannelVersionID], row)
	}
	return result, nil
}

func applyChannelFilters(db *gorm.DB, req amazonReq.LogisticsChannelPageReq, tableName, batchTable string) *gorm.DB {
	if provider := strings.TrimSpace(req.Provider); provider != "" {
		db = db.Where(tableName+".provider = ?", strings.ToLower(provider))
	}

	switch strings.ToLower(strings.TrimSpace(req.ActiveScope)) {
	case "", "current":
		db = db.Where(tableName+".is_active = ?", true)
	case "history":
		db = db.Where(tableName+".is_active = ?", false)
	case "all":
	}

	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		db = db.Where(
			tableName+".product_code LIKE ? OR "+tableName+".channel_name LIKE ? OR "+tableName+".sheet_name LIKE ? OR "+tableName+".country_label LIKE ? OR "+batchTable+".source_file_name LIKE ?",
			pattern, pattern, pattern, pattern, pattern,
		)
	}

	if logisticsProvider := strings.TrimSpace(req.LogisticsProvider); logisticsProvider != "" {
		db = db.Where(tableName+".logistics_provider LIKE ?", "%"+logisticsProvider+"%")
	}

	if countryLabel := strings.TrimSpace(req.CountryLabel); countryLabel != "" {
		db = db.Where(tableName+".country_label LIKE ?", "%"+countryLabel+"%")
	}

	if platform := normalizeLogisticsPlatformFilter(req.Platform); platform != "" {
		db = applyLogisticsPlatformFilter(db, tableName, platform)
	}

	if start, ok := parseFlexibleTime(req.EffectiveDateFrom); ok {
		db = db.Where(tableName+".effective_at >= ?", start)
	}
	if end, ok := parseFlexibleTime(req.EffectiveDateTo); ok {
		db = db.Where(tableName+".effective_at <= ?", end)
	}
	if start, ok := parseFlexibleTime(req.UploadedDateFrom); ok {
		db = db.Where(batchTable+".created_at >= ?", start)
	}
	if end, ok := parseFlexibleTime(req.UploadedDateTo); ok {
		db = db.Where(batchTable+".created_at <= ?", end)
	}

	return db
}

func applyLogisticsPlatformFilter(db *gorm.DB, tableName, platform string) *gorm.DB {
	if platform == logisticsPlatformAll {
		db = db.Where("("+tableName+".platform = ? OR "+tableName+".platform = '' OR "+tableName+".platform IS NULL)", logisticsPlatformAll)
		for _, keyword := range allLogisticsPlatformKeywords() {
			pattern := "%" + keyword + "%"
			db = db.Where(tableName+".channel_name NOT LIKE ? AND "+tableName+".sheet_name NOT LIKE ? AND "+tableName+".logistics_provider NOT LIKE ?", pattern, pattern, pattern)
		}
		return db
	}

	conditions := []string{tableName + ".platform = ?"}
	args := []interface{}{platform}
	for _, keyword := range logisticsPlatformKeywords(platform) {
		pattern := "%" + keyword + "%"
		conditions = append(conditions, tableName+".channel_name LIKE ? OR "+tableName+".sheet_name LIKE ? OR "+tableName+".logistics_provider LIKE ?")
		args = append(args, pattern, pattern, pattern)
	}
	return db.Where("("+strings.Join(conditions, " OR ")+")", args...)
}

func buildChannelVersionRecord(batchID uint, provider string, channel logisticsChannel, now time.Time) amazonModel.LogisticsChannelVersion {
	productCode := strings.TrimSpace(channel.ServiceCode)
	productCodeType := strings.TrimSpace(channel.ServiceCodeType)
	logicalKey := strings.TrimSpace(channel.LogicalProductKey)
	if logicalKey == "" {
		logicalKey = productCode
	}
	warnings := append([]string{}, channel.Warnings...)

	if logicalKey == "" {
		logicalKey = normalizedText(channel.SheetName)
		if logicalKey == "" {
			logicalKey = normalizedText(channel.ChannelName)
		}
		productCodeType = defaultString(productCodeType, "sheet_name_fallback")
		warnings = append(warnings, "未识别产品代码，使用sheet名称作为逻辑产品键")
	} else {
		logicalKey = normalizedText(logicalKey)
	}

	return amazonModel.LogisticsChannelVersion{
		BatchID:             batchID,
		Provider:            provider,
		LogicalProductKey:   logicalKey,
		ProductCode:         productCode,
		ProductCodeType:     productCodeType,
		ChannelName:         channel.ChannelName,
		SheetName:           channel.SheetName,
		LogisticsProvider:   channel.LogisticsProvider,
		Platform:            displayLogisticsPlatform(channel.Platform, channel.LogisticsProvider, channel.ChannelName, channel.SheetName),
		ServiceCode:         channel.ServiceCode,
		EffectiveAt:         channel.EffectiveAt,
		EffectiveTextRaw:    channel.EffectiveTextRaw,
		TransitTime:         compactTransitTime(channel.TransitTime),
		CountryLabel:        channel.CountryLabel,
		SupportsBattery:     channel.SupportsBattery,
		RequiresBattery:     channel.RequiresBattery,
		RateKind:            channel.RateKind,
		VolumeDivisor:       channel.VolumeDivisor,
		VolumeThreshold:     channel.VolumeThreshold,
		VolumeThresholdMax:  channel.VolumeThresholdMax,
		IgnoreVolumetric:    channel.IgnoreVolumetric,
		MinBillableWeightKG: channel.MinBillableWeightKG,
		StepWeightKG:        channel.StepWeightKG,
		SizeRulesJSON:       mustMarshalJSON(channel.SizeRules),
		TagsJSON:            mustMarshalJSON(uniqueStrings(channel.Tags)),
		WarningsJSON:        mustMarshalJSON(uniqueStrings(warnings)),
		UnresolvedFeesJSON:  mustMarshalJSON(uniqueStrings(channel.UnresolvedFees)),
		ZoneBased:           channel.ZoneBased,
		IsActive:            true,
		ActivatedAt:         &now,
	}
}

func buildChannelFromVersion(version amazonModel.LogisticsChannelVersion, rows []amazonModel.LogisticsRateRowVersion) (logisticsChannel, error) {
	sizeRules, err := decodeSizeRulesJSON(version.SizeRulesJSON)
	if err != nil {
		return logisticsChannel{}, err
	}
	tags, err := decodeStringSliceJSON(version.TagsJSON)
	if err != nil {
		return logisticsChannel{}, err
	}
	warnings, err := decodeStringSliceJSON(version.WarningsJSON)
	if err != nil {
		return logisticsChannel{}, err
	}
	unresolvedFees, err := decodeStringSliceJSON(version.UnresolvedFeesJSON)
	if err != nil {
		return logisticsChannel{}, err
	}

	rateRows := make([]logisticsRateRow, 0, len(rows))
	for _, row := range rows {
		rateRows = append(rateRows, logisticsRateRow{
			Zone:               row.Zone,
			WeightMinKG:        row.WeightMinKG,
			WeightMaxKG:        row.WeightMaxKG,
			RatePerKG:          row.RatePerKG,
			HandlingFeeCNY:     row.HandlingFeeCNY,
			RegistrationFeeCNY: row.RegistrationFeeCNY,
			FirstWeightKG:      row.FirstWeightKG,
			FirstPriceCNY:      row.FirstPriceCNY,
			ContinueWeightKG:   row.ContinueWeightKG,
			ContinuePriceCNY:   row.ContinuePriceCNY,
			MinBillableWeight:  row.MinBillableWeight,
			TransitTime:        compactTransitTime(row.TransitTime),
			VolumeRatioMin:     row.VolumeRatioMin,
			VolumeRatioMax:     row.VolumeRatioMax,
			BillableWeightMode: row.BillableWeightMode,
			RateLabelRaw:       row.RateLabelRaw,
		})
	}

	return logisticsChannel{
		ChannelVersionID:    version.ID,
		Provider:            version.Provider,
		LogicalProductKey:   version.LogicalProductKey,
		LogisticsProvider:   version.LogisticsProvider,
		Platform:            displayLogisticsPlatform(version.Platform, version.LogisticsProvider, version.ChannelName, version.SheetName),
		ChannelName:         version.ChannelName,
		SheetName:           version.SheetName,
		ServiceCode:         version.ServiceCode,
		ServiceCodeType:     version.ProductCodeType,
		TransitTime:         compactTransitTime(version.TransitTime),
		CountryLabel:        version.CountryLabel,
		EffectiveAt:         version.EffectiveAt,
		EffectiveTextRaw:    version.EffectiveTextRaw,
		Tags:                tags,
		Warnings:            warnings,
		UnresolvedFees:      unresolvedFees,
		SupportsBattery:     version.SupportsBattery,
		RequiresBattery:     version.RequiresBattery,
		RateKind:            version.RateKind,
		Rows:                rateRows,
		VolumeDivisor:       version.VolumeDivisor,
		VolumeThreshold:     version.VolumeThreshold,
		VolumeThresholdMax:  version.VolumeThresholdMax,
		IgnoreVolumetric:    version.IgnoreVolumetric,
		MinBillableWeightKG: version.MinBillableWeightKG,
		StepWeightKG:        version.StepWeightKG,
		SizeRules:           sizeRules,
		ZoneBased:           version.ZoneBased,
	}, nil
}

func mustMarshalJSON(value interface{}) datatypes.JSON {
	raw, err := json.Marshal(value)
	if err != nil || len(raw) == 0 {
		return datatypes.JSON([]byte("null"))
	}
	return datatypes.JSON(raw)
}

func decodeStringSliceJSON(raw datatypes.JSON) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func decodeSizeRulesJSON(raw datatypes.JSON) (logisticsSizeRules, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return logisticsSizeRules{}, nil
	}
	var rules logisticsSizeRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return logisticsSizeRules{}, err
	}
	return rules, nil
}

func collectChannelVersionIDs(versions []amazonModel.LogisticsChannelVersion) []uint {
	ids := make([]uint, 0, len(versions))
	for _, version := range versions {
		ids = append(ids, version.ID)
	}
	return ids
}

func mapKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}

func normalizePageInfo(page, pageSize int) amazonReq.LogisticsChannelPageReq {
	req := amazonReq.LogisticsChannelPageReq{}
	req.Page = page
	req.PageSize = pageSize
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	return req
}
