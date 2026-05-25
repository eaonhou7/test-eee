package amazon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"github.com/xuri/excelize/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SupportInboxService struct{}

var supportTemplateVariablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

func (s *SupportInboxService) List(ctx context.Context, req amazonReq.AmazonSupportCaseListReq) (SupportCasePageResult, error) {
	base := s.buildSupportCaseBaseQuery(ctx, req)
	summary, err := s.buildSupportSummary(base.Session(&gorm.Session{}))
	if err != nil {
		return SupportCasePageResult{}, err
	}
	db := s.applySupportCaseFilters(base.Session(&gorm.Session{}), req)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return SupportCasePageResult{}, err
	}
	now := time.Now().UTC()
	warningAt := now.Add(supportWarningLeadHours * time.Hour)
	var rows []amazonModel.SupportCase
	if err := db.
		Order(supportCaseListOrder(now, warningAt)).
		Scopes(req.PageInfo.Paginate()).
		Find(&rows).Error; err != nil {
		return SupportCasePageResult{}, err
	}
	items, err := s.mapSupportCaseListItems(ctx, rows)
	if err != nil {
		return SupportCasePageResult{}, err
	}
	return SupportCasePageResult{
		List:     items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Summary:  summary,
	}, nil
}

func supportCaseListOrder(now, warningAt time.Time) clause.OrderBy {
	return clause.OrderBy{Expression: clause.Expr{
		SQL: strings.Join([]string{
			"CASE WHEN read_status = ? THEN 0 ELSE 1 END",
			"CASE WHEN handling_status <> ? AND due_at IS NOT NULL AND due_at < ? THEN 0 ELSE 1 END",
			"CASE WHEN handling_status <> ? AND due_at IS NOT NULL AND due_at >= ? AND due_at <= ? THEN 0 ELSE 1 END",
			"COALESCE(last_customer_at, first_received_at) DESC",
			"id DESC",
		}, ", "),
		Vars: []interface{}{
			supportReadStatusUnread,
			supportHandlingStatusClosed, now,
			supportHandlingStatusClosed, now, warningAt,
		},
	}}
}

func (s *SupportInboxService) Find(ctx context.Context, id uint) (SupportCaseDetail, error) {
	if id == 0 {
		return SupportCaseDetail{}, errors.New("id is required")
	}
	var row amazonModel.SupportCase
	if err := global.GVA_DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return SupportCaseDetail{}, err
	}
	return s.buildSupportCaseDetail(ctx, row)
}

func (s *SupportInboxService) UpsertCase(ctx context.Context, req amazonReq.AmazonSupportCaseUpsertReq) (SupportCaseDetail, error) {
	if req.StoreID == 0 && req.OrderID == nil && req.ReturnOrderID == nil {
		return SupportCaseDetail{}, errors.New("storeId or related order is required")
	}
	var caseID uint
	err := global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, _, err := s.saveSupportCaseTx(ctx, tx, req, false)
		if err != nil {
			return err
		}
		caseID = row.ID
		return nil
	})
	if err != nil {
		return SupportCaseDetail{}, err
	}
	return s.Find(ctx, caseID)
}

func (s *SupportInboxService) MarkRead(ctx context.Context, id uint) (SupportCaseDetail, error) {
	if id == 0 {
		return SupportCaseDetail{}, errors.New("id is required")
	}
	if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.SupportCase{}).Where("id = ?", id).Updates(map[string]interface{}{
		"read_status": supportReadStatusRead,
	}).Error; err != nil {
		return SupportCaseDetail{}, err
	}
	return s.Find(ctx, id)
}

func (s *SupportInboxService) MarkPending(ctx context.Context, id uint) (SupportCaseDetail, error) {
	if id == 0 {
		return SupportCaseDetail{}, errors.New("id is required")
	}
	if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.SupportCase{}).Where("id = ?", id).Updates(map[string]interface{}{
		"read_status":     supportReadStatusRead,
		"handling_status": supportHandlingStatusPending,
	}).Error; err != nil {
		return SupportCaseDetail{}, err
	}
	return s.Find(ctx, id)
}

func (s *SupportInboxService) Close(ctx context.Context, id uint) (SupportCaseDetail, error) {
	if id == 0 {
		return SupportCaseDetail{}, errors.New("id is required")
	}
	if err := global.GVA_DB.WithContext(ctx).Model(&amazonModel.SupportCase{}).Where("id = ?", id).Updates(map[string]interface{}{
		"read_status":     supportReadStatusRead,
		"handling_status": supportHandlingStatusClosed,
	}).Error; err != nil {
		return SupportCaseDetail{}, err
	}
	return s.Find(ctx, id)
}

func (s *SupportInboxService) RefreshActions(ctx context.Context, caseID uint) ([]SupportActionAvailability, error) {
	if caseID == 0 {
		return nil, errors.New("caseId is required")
	}
	row, order, store, err := s.loadSupportActionContext(ctx, caseID)
	if err != nil {
		return nil, err
	}
	if order.ID == 0 || store.ID == 0 {
		now := time.Now().UTC()
		_ = global.GVA_DB.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
			"is_direct_send_available": false,
			"last_action_sync_at":      &now,
		}).Error
		return nil, nil
	}
	actions, payload, err := new(AmazonMessagingService).GetMessagingActions(ctx, store, order)
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"is_direct_send_available": len(actions) > 0,
		"last_action_sync_at":      &now,
	}
	rawSource := decodeJSONMap(row.RawSourceJSON)
	rawSource["messagingActions"] = actions
	if payload != nil {
		rawSource["messagingActionPayload"] = payload
	}
	updates["raw_source_json"] = encodeJSONObject(rawSource)
	_ = global.GVA_DB.WithContext(ctx).Model(&row).Updates(updates).Error
	if err != nil {
		return nil, err
	}
	return actions, nil
}

func (s *SupportInboxService) SendReply(ctx context.Context, req amazonReq.AmazonSupportSendReplyReq) (SupportReplyResult, error) {
	if req.CaseID == 0 {
		return SupportReplyResult{}, errors.New("caseId is required")
	}
	templateDetail, err := new(SupportTemplateService).Find(ctx, req.TemplateID)
	if err != nil {
		return SupportReplyResult{}, err
	}
	detail, err := s.Find(ctx, req.CaseID)
	if err != nil {
		return SupportReplyResult{}, err
	}
	variables := s.defaultSupportTemplateVariables(detail)
	for key, value := range req.Variables {
		variables[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	renderedSubject := renderSupportTemplate(templateDetail.SubjectTemplate, variables)
	renderedBody := renderSupportTemplate(templateDetail.BodyTemplate, variables)
	deliveryMode := normalizeSupportDeliveryMode(defaultString(req.DeliveryMode, templateDetail.DeliveryMode))

	result := SupportReplyResult{
		CaseID:          detail.ID,
		DeliveryMode:    deliveryMode,
		RenderedSubject: renderedSubject,
		RenderedBody:    renderedBody,
	}

	err = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row amazonModel.SupportCase
		if err := tx.First(&row, detail.ID).Error; err != nil {
			return err
		}
		message := amazonModel.SupportCaseMessage{
			CaseID:      row.ID,
			Role:        supportMessageRoleAgent,
			TemplateKey: templateDetail.Code,
			BodyPlain:   renderedBody,
			SentAt:      timePtr(time.Now().UTC()),
		}
		if deliveryMode == supportDeliveryModeAmazonDirect {
			action, sendStatus, sendErr := s.sendSupportDirectMessage(ctx, row, templateDetail, req, renderedSubject, renderedBody)
			if action != nil {
				result.Action = action
				message.ExternalActionKey = action.ActionKey
			}
			message.SendStatus = sendStatus
			message.Channel = supportMessageChannelAmazon
			if sendStatus == supportSendStatusFallbackManual {
				message.Channel = supportMessageChannelManualCopy
			}
			if sendErr != nil {
				message.ErrorMessage = sendErr.Error()
				if sendStatus == supportSendStatusFailed {
					result.SendStatus = supportSendStatusFailed
				} else {
					result.SendStatus = supportSendStatusFallbackManual
				}
			} else {
				result.SendStatus = sendStatus
			}
		} else {
			message.SendStatus = supportSendStatusCopied
			message.Channel = supportMessageChannelManualCopy
			result.SendStatus = supportSendStatusCopied
		}
		if message.SendStatus == "" {
			message.SendStatus = supportSendStatusCopied
		}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		result.MessageID = message.ID
		now := time.Now().UTC()
		updates := map[string]interface{}{
			"read_status":     supportReadStatusRead,
			"handling_status": supportHandlingStatusProcessing,
			"last_agent_at":   &now,
			"latest_excerpt":  truncateSupportExcerpt(renderedBody),
			"updated_at":      &now,
		}
		if row.HandlingStatus == supportHandlingStatusClosed {
			updates["handling_status"] = supportHandlingStatusClosed
		}
		return tx.Model(&row).Updates(updates).Error
	})
	if err != nil {
		return SupportReplyResult{}, err
	}
	return result, nil
}

func (s *SupportInboxService) ImportWorkbook(ctx context.Context, fileName string, raw []byte) (SupportImportResult, error) {
	job := amazonModel.SupportImportJob{
		FileName: defaultString(strings.TrimSpace(fileName), "support-inbox.xlsx"),
		Status:   "processing",
	}
	if err := global.GVA_DB.WithContext(ctx).Create(&job).Error; err != nil {
		return SupportImportResult{}, err
	}
	result := SupportImportResult{
		JobID:    job.ID,
		FileName: job.FileName,
	}
	errorsList := make([]SupportImportErrorItem, 0)
	file, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return s.finishSupportImportJob(ctx, job, result, errorsList, err)
	}
	defer file.Close()
	sheetName := file.GetSheetName(0)
	if sheetName == "" {
		return s.finishSupportImportJob(ctx, job, result, errorsList, errors.New("工作簿没有可用工作表"))
	}
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return s.finishSupportImportJob(ctx, job, result, errorsList, err)
	}
	if len(rows) < 2 {
		return s.finishSupportImportJob(ctx, job, result, errorsList, errors.New("工作簿缺少数据行"))
	}
	headerMap := supportWorkbookHeaderMap(rows[0])
	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		if supportWorkbookRowEmpty(row) {
			continue
		}
		result.TotalRows++
		req, buildErr := s.buildSupportImportUpsertReq(ctx, headerMap, row)
		if buildErr != nil {
			errorsList = append(errorsList, SupportImportErrorItem{Row: rowIndex + 1, Message: buildErr.Error()})
			continue
		}
		buildErr = global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			_, _, err := s.saveSupportCaseTx(ctx, tx, req, true)
			return err
		})
		if buildErr != nil {
			errorsList = append(errorsList, SupportImportErrorItem{Row: rowIndex + 1, Message: buildErr.Error()})
			continue
		}
		result.SuccessRows++
	}
	result.FailedRows = len(errorsList)
	return s.finishSupportImportJob(ctx, job, result, errorsList, nil)
}

func (s *SupportInboxService) finishSupportImportJob(ctx context.Context, job amazonModel.SupportImportJob, result SupportImportResult, errorsList []SupportImportErrorItem, processErr error) (SupportImportResult, error) {
	result.Errors = errorsList
	result.FailedRows = len(errorsList)
	status := "success"
	if processErr != nil {
		status = "failed"
		errorsList = append(errorsList, SupportImportErrorItem{Row: 0, Message: processErr.Error()})
		result.Errors = errorsList
		result.FailedRows = len(errorsList)
	}
	if result.SuccessRows > 0 && result.FailedRows > 0 {
		status = "partial_success"
	}
	now := time.Now().UTC()
	_ = global.GVA_DB.WithContext(ctx).Model(&job).Updates(map[string]interface{}{
		"status":            status,
		"total_rows":        result.TotalRows,
		"success_rows":      result.SuccessRows,
		"failed_rows":       result.FailedRows,
		"error_report_json": encodeJSON(errorsList),
		"finished_at":       &now,
	}).Error
	if processErr != nil {
		return result, processErr
	}
	return result, nil
}

func (s *SupportInboxService) buildSupportImportUpsertReq(ctx context.Context, headers map[string]int, row []string) (amazonReq.AmazonSupportCaseUpsertReq, error) {
	storeName := strings.TrimSpace(supportWorkbookCell(headers, row, "store_name"))
	if storeName == "" {
		return amazonReq.AmazonSupportCaseUpsertReq{}, errors.New("store_name 不能为空")
	}
	var store amazonModel.StoreAccount
	if err := global.GVA_DB.WithContext(ctx).Where("store_name = ?", storeName).First(&store).Error; err != nil {
		return amazonReq.AmazonSupportCaseUpsertReq{}, fmt.Errorf("店铺不存在: %s", storeName)
	}
	caseType := normalizeSupportCaseType(supportWorkbookCell(headers, row, "case_type"))
	if caseType == "" {
		caseType = supportCaseTypeBuyerMessage
	}
	subject := supportWorkbookCell(headers, row, "subject")
	messageBody := supportWorkbookCell(headers, row, "message_body")
	receivedAt := supportWorkbookCell(headers, row, "received_at")
	amazonOrderID := supportWorkbookCell(headers, row, "amazon_order_id")
	amazonRMAID := supportWorkbookCell(headers, row, "amazon_rma_id")
	siteCode := strings.TrimSpace(supportWorkbookCell(headers, row, "site_code"))
	var orderID *uint
	if amazonOrderID != "" {
		var order amazonModel.Order
		if err := global.GVA_DB.WithContext(ctx).Where("store_id = ? AND amazon_order_id = ?", store.ID, amazonOrderID).First(&order).Error; err == nil {
			orderID = &order.ID
			siteCode = defaultString(siteCode, order.SiteCode)
		}
	}
	var returnOrderID *uint
	if amazonRMAID != "" {
		var returnOrder amazonModel.ReturnOrder
		if err := global.GVA_DB.WithContext(ctx).Where("store_id = ? AND amazon_rma_id = ?", store.ID, amazonRMAID).First(&returnOrder).Error; err == nil {
			returnOrderID = &returnOrder.ID
			siteCode = defaultString(siteCode, returnOrder.SiteCode)
			if orderID == nil && returnOrder.OrderID != nil {
				orderID = returnOrder.OrderID
			}
		}
	}
	return amazonReq.AmazonSupportCaseUpsertReq{
		StoreID:         store.ID,
		SiteCode:        siteCode,
		CaseType:        caseType,
		SourceType:      supportSourceTypeImport,
		SourceRefType:   "workbook",
		OrderID:         orderID,
		ReturnOrderID:   returnOrderID,
		ExternalCaseID:  supportWorkbookCell(headers, row, "external_case_id"),
		Subject:         defaultString(subject, truncateSupportExcerpt(messageBody)),
		BuyerName:       supportWorkbookCell(headers, row, "buyer_name"),
		BuyerEmail:      supportWorkbookCell(headers, row, "buyer_email"),
		FirstReceivedAt: receivedAt,
		MessageBody:     messageBody,
		Notes:           supportWorkbookCell(headers, row, "notes"),
		RawSource: map[string]interface{}{
			"importSource":  "workbook",
			"amazonOrderId": amazonOrderID,
			"amazonRmaId":   amazonRMAID,
		},
	}, nil
}

func (s *SupportInboxService) buildSupportCaseBaseQuery(ctx context.Context, req amazonReq.AmazonSupportCaseListReq) *gorm.DB {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.SupportCase{})
	if req.StoreID > 0 {
		db = db.Where("store_id = ?", req.StoreID)
	}
	if strings.TrimSpace(req.SiteCode) != "" {
		db = db.Where("site_code = ?", strings.TrimSpace(req.SiteCode))
	}
	if strings.TrimSpace(req.CaseType) != "" {
		db = db.Where("case_type = ?", normalizeSupportCaseType(req.CaseType))
	}
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("subject LIKE ? OR buyer_name LIKE ? OR buyer_email LIKE ? OR external_case_id LIKE ? OR latest_excerpt LIKE ?", keyword, keyword, keyword, keyword, keyword)
	}
	return db
}

func (s *SupportInboxService) applySupportCaseFilters(db *gorm.DB, req amazonReq.AmazonSupportCaseListReq) *gorm.DB {
	now := time.Now().UTC()
	warningAt := now.Add(supportWarningLeadHours * time.Hour)
	if strings.TrimSpace(req.ReadStatus) != "" {
		db = db.Where("read_status = ?", normalizeSupportReadStatus(req.ReadStatus))
	}
	if strings.TrimSpace(req.HandlingStatus) != "" {
		db = db.Where("handling_status = ?", normalizeSupportHandlingStatus(req.HandlingStatus))
	}
	switch strings.TrimSpace(req.SLABucket) {
	case supportSLABucketOverdue:
		db = db.Where("handling_status <> ? AND due_at IS NOT NULL AND due_at < ?", supportHandlingStatusClosed, now)
	case supportSLABucketWarning:
		db = db.Where("handling_status <> ? AND due_at IS NOT NULL AND due_at >= ? AND due_at <= ?", supportHandlingStatusClosed, now, warningAt)
	case supportSLABucketNormal:
		db = db.Where("(due_at IS NULL OR due_at > ?) AND handling_status <> ?", warningAt, supportHandlingStatusClosed)
	}
	return db
}

func (s *SupportInboxService) buildSupportSummary(db *gorm.DB) (SupportInboxSummary, error) {
	now := time.Now().UTC()
	warningAt := now.Add(supportWarningLeadHours * time.Hour)
	summary := SupportInboxSummary{}
	if err := db.Count(&summary.AllCount).Error; err != nil {
		return summary, err
	}
	if err := db.Session(&gorm.Session{}).Where("read_status = ?", supportReadStatusUnread).Count(&summary.UnreadCount).Error; err != nil {
		return summary, err
	}
	if err := db.Session(&gorm.Session{}).Where("handling_status = ?", supportHandlingStatusPending).Count(&summary.PendingCount).Error; err != nil {
		return summary, err
	}
	if err := db.Session(&gorm.Session{}).Where("handling_status <> ? AND due_at IS NOT NULL AND due_at < ?", supportHandlingStatusClosed, now).Count(&summary.OverdueCount).Error; err != nil {
		return summary, err
	}
	if err := db.Session(&gorm.Session{}).Where("handling_status <> ? AND due_at IS NOT NULL AND due_at >= ? AND due_at <= ?", supportHandlingStatusClosed, now, warningAt).Count(&summary.WarningCount).Error; err != nil {
		return summary, err
	}
	return summary, nil
}

func (s *SupportInboxService) mapSupportCaseListItems(ctx context.Context, rows []amazonModel.SupportCase) ([]SupportCaseListItem, error) {
	if len(rows) == 0 {
		return []SupportCaseListItem{}, nil
	}
	storeMap, orderMap, returnMap, err := loadSupportReferenceMaps(ctx, rows)
	if err != nil {
		return nil, err
	}
	result := make([]SupportCaseListItem, 0, len(rows))
	now := time.Now().UTC()
	for _, row := range rows {
		store := storeMap[row.StoreID]
		order := orderMap[valueOrZeroUint(row.OrderID)]
		returnOrder := returnMap[valueOrZeroUint(row.ReturnOrderID)]
		slaBucket, remaining := supportSLAMetrics(row.DueAt, row.HandlingStatus, now)
		item := SupportCaseListItem{
			ID:                    row.ID,
			StoreID:               row.StoreID,
			StoreName:             store.StoreName,
			SiteCode:              row.SiteCode,
			CaseType:              row.CaseType,
			SourceType:            row.SourceType,
			OrderID:               row.OrderID,
			ReturnOrderID:         row.ReturnOrderID,
			AmazonOrderID:         order.AmazonOrderID,
			AmazonRMAID:           returnOrder.AmazonRMAID,
			ExternalCaseID:        row.ExternalCaseID,
			Subject:               row.Subject,
			BuyerName:             row.BuyerName,
			BuyerEmail:            row.BuyerEmail,
			FirstReceivedAt:       formatCollectorTime(row.FirstReceivedAt),
			DueAt:                 formatCollectorTime(row.DueAt),
			ReadStatus:            row.ReadStatus,
			HandlingStatus:        row.HandlingStatus,
			LastCustomerAt:        formatCollectorTime(row.LastCustomerAt),
			LastAgentAt:           formatCollectorTime(row.LastAgentAt),
			LatestExcerpt:         row.LatestExcerpt,
			IsDirectSendAvailable: row.IsDirectSendAvailable,
			LastActionSyncAt:      formatCollectorTime(row.LastActionSyncAt),
			SLABucket:             slaBucket,
			RemainingMinutes:      remaining,
		}
		if item.AmazonOrderID == "" && returnOrder.AmazonOrderID != "" {
			item.AmazonOrderID = returnOrder.AmazonOrderID
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *SupportInboxService) buildSupportCaseDetail(ctx context.Context, row amazonModel.SupportCase) (SupportCaseDetail, error) {
	storeMap, orderMap, returnMap, err := loadSupportReferenceMaps(ctx, []amazonModel.SupportCase{row})
	if err != nil {
		return SupportCaseDetail{}, err
	}
	var messages []amazonModel.SupportCaseMessage
	if err := global.GVA_DB.WithContext(ctx).Where("case_id = ?", row.ID).Order("COALESCE(sent_at, created_at) ASC").Order("id ASC").Find(&messages).Error; err != nil {
		return SupportCaseDetail{}, err
	}
	store := storeMap[row.StoreID]
	order := orderMap[valueOrZeroUint(row.OrderID)]
	returnOrder := returnMap[valueOrZeroUint(row.ReturnOrderID)]
	var orderContext *OrderDetail
	var returnContext *ReturnOrderDetail
	if row.OrderID != nil {
		detail, err := new(OrderService).findDetail(ctx, *row.OrderID, true)
		if err == nil {
			orderContext = &detail
		}
	}
	if row.ReturnOrderID != nil {
		detail, err := new(ReturnService).Find(ctx, *row.ReturnOrderID)
		if err == nil {
			returnContext = &detail
			if orderContext == nil && detail.OrderID != nil {
				if orderDetail, err := new(OrderService).findDetail(ctx, *detail.OrderID, true); err == nil {
					orderContext = &orderDetail
				}
			}
		}
	}
	slaBucket, remaining := supportSLAMetrics(row.DueAt, row.HandlingStatus, time.Now().UTC())
	detail := SupportCaseDetail{
		ID:                    row.ID,
		StoreID:               row.StoreID,
		StoreName:             store.StoreName,
		SiteCode:              row.SiteCode,
		CaseType:              row.CaseType,
		SourceType:            row.SourceType,
		SourceRefType:         row.SourceRefType,
		SourceRefID:           row.SourceRefID,
		OrderID:               row.OrderID,
		ReturnOrderID:         row.ReturnOrderID,
		AmazonOrderID:         order.AmazonOrderID,
		AmazonRMAID:           returnOrder.AmazonRMAID,
		ExternalCaseID:        row.ExternalCaseID,
		Subject:               row.Subject,
		BuyerName:             row.BuyerName,
		BuyerEmail:            row.BuyerEmail,
		FirstReceivedAt:       formatCollectorTime(row.FirstReceivedAt),
		DueAt:                 formatCollectorTime(row.DueAt),
		ReadStatus:            row.ReadStatus,
		HandlingStatus:        row.HandlingStatus,
		LastCustomerAt:        formatCollectorTime(row.LastCustomerAt),
		LastAgentAt:           formatCollectorTime(row.LastAgentAt),
		LatestExcerpt:         row.LatestExcerpt,
		IsDirectSendAvailable: row.IsDirectSendAvailable,
		LastActionSyncAt:      formatCollectorTime(row.LastActionSyncAt),
		SLABucket:             slaBucket,
		RemainingMinutes:      remaining,
		RawSource:             decodeJSONMap(row.RawSourceJSON),
		Messages:              make([]SupportMessageDetail, 0, len(messages)),
		ActionAvailability:    decodeSupportActionAvailability(row.RawSourceJSON),
		OrderContext:          orderContext,
		ReturnContext:         returnContext,
	}
	if detail.AmazonOrderID == "" && returnOrder.AmazonOrderID != "" {
		detail.AmazonOrderID = returnOrder.AmazonOrderID
	}
	for _, message := range messages {
		detail.Messages = append(detail.Messages, SupportMessageDetail{
			ID:                message.ID,
			CaseID:            message.CaseID,
			Role:              message.Role,
			Channel:           message.Channel,
			TemplateKey:       message.TemplateKey,
			BodyPlain:         message.BodyPlain,
			SendStatus:        message.SendStatus,
			ExternalActionKey: message.ExternalActionKey,
			ExternalMessageID: message.ExternalMessageID,
			RawPayload:        decodeJSONMap(message.RawPayloadJSON),
			SentAt:            formatCollectorTime(message.SentAt),
			ErrorMessage:      message.ErrorMessage,
			CreatedAt:         formatCollectorTime(&message.CreatedAt),
		})
	}
	return detail, nil
}

func (s *SupportInboxService) loadSupportActionContext(ctx context.Context, caseID uint) (amazonModel.SupportCase, amazonModel.Order, amazonModel.StoreAccount, error) {
	var row amazonModel.SupportCase
	if err := global.GVA_DB.WithContext(ctx).First(&row, caseID).Error; err != nil {
		return row, amazonModel.Order{}, amazonModel.StoreAccount{}, err
	}
	var order amazonModel.Order
	if row.OrderID != nil {
		_ = global.GVA_DB.WithContext(ctx).First(&order, *row.OrderID).Error
	}
	if order.ID == 0 && row.ReturnOrderID != nil {
		var returnOrder amazonModel.ReturnOrder
		if err := global.GVA_DB.WithContext(ctx).First(&returnOrder, *row.ReturnOrderID).Error; err == nil && returnOrder.OrderID != nil {
			_ = global.GVA_DB.WithContext(ctx).First(&order, *returnOrder.OrderID).Error
		}
	}
	var store amazonModel.StoreAccount
	if row.StoreID > 0 {
		_ = global.GVA_DB.WithContext(ctx).First(&store, row.StoreID).Error
	}
	return row, order, store, nil
}

func (s *SupportInboxService) sendSupportDirectMessage(ctx context.Context, row amazonModel.SupportCase, template SupportTemplateDetail, req amazonReq.AmazonSupportSendReplyReq, renderedSubject, renderedBody string) (*SupportActionAvailability, string, error) {
	_, order, store, err := s.loadSupportActionContext(ctx, row.ID)
	if err != nil {
		return nil, supportSendStatusFallbackManual, err
	}
	if order.ID == 0 || store.ID == 0 || strings.TrimSpace(store.AuthStatus) != "authorized" {
		return nil, supportSendStatusFallbackManual, errors.New("当前案例未关联可授权直发的 Amazon 订单")
	}
	actions, err := s.RefreshActions(ctx, row.ID)
	if err != nil {
		return nil, supportSendStatusFailed, err
	}
	actionKey := normalizeSupportActionKey(defaultString(req.ActionKey, template.AmazonActionKey))
	var matched *SupportActionAvailability
	for _, action := range actions {
		action := action
		if normalizeSupportActionKey(action.ActionKey) == actionKey {
			matched = &action
			break
		}
		if req.ActionPath != "" && strings.TrimSpace(action.Path) == strings.TrimSpace(req.ActionPath) {
			matched = &action
		}
	}
	if matched == nil {
		return nil, supportSendStatusFallbackManual, errors.New("Amazon 当前未返回与模板匹配的直发动作")
	}
	if _, err := new(AmazonMessagingService).SendMessage(ctx, store, order, *matched, renderedBody); err != nil {
		return matched, supportSendStatusFailed, err
	}
	return matched, supportSendStatusSent, nil
}

func (s *SupportInboxService) defaultSupportTemplateVariables(detail SupportCaseDetail) map[string]string {
	result := map[string]string{
		"buyer_name":       detail.BuyerName,
		"buyer_email":      detail.BuyerEmail,
		"amazon_order_id":  detail.AmazonOrderID,
		"amazon_rma_id":    detail.AmazonRMAID,
		"store_name":       detail.StoreName,
		"site_code":        detail.SiteCode,
		"case_subject":     detail.Subject,
		"external_case_id": detail.ExternalCaseID,
	}
	if detail.OrderContext != nil {
		result["order_status"] = detail.OrderContext.OrderStatus
		result["fulfillment_type"] = detail.OrderContext.FulfillmentType
		result["tracking_no"] = ""
		if len(detail.OrderContext.Shipments) > 0 {
			result["tracking_no"] = detail.OrderContext.Shipments[0].TrackingNo
		}
	}
	if detail.ReturnContext != nil {
		result["return_status"] = detail.ReturnContext.ReturnRequestStatus
		result["return_tracking_id"] = detail.ReturnContext.TrackingID
	}
	return result
}

func (s *SupportInboxService) saveSupportCaseTx(ctx context.Context, tx *gorm.DB, req amazonReq.AmazonSupportCaseUpsertReq, importMode bool) (amazonModel.SupportCase, *amazonModel.SupportCaseMessage, error) {
	resolved, err := s.resolveSupportAssociationsTx(ctx, tx, req)
	if err != nil {
		return amazonModel.SupportCase{}, nil, err
	}
	var row amazonModel.SupportCase
	if req.ID > 0 {
		if err := tx.First(&row, req.ID).Error; err != nil {
			return amazonModel.SupportCase{}, nil, err
		}
	} else if strings.TrimSpace(req.ExternalCaseID) != "" && resolved.Store.ID > 0 {
		_ = tx.Where("store_id = ? AND external_case_id = ?", resolved.Store.ID, strings.TrimSpace(req.ExternalCaseID)).First(&row).Error
	}
	firstReceivedAt := parseSupportTime(req.FirstReceivedAt)
	if firstReceivedAt == nil {
		firstReceivedAt = timePtr(time.Now().UTC())
	}
	if row.FirstReceivedAt != nil {
		firstReceivedAt = row.FirstReceivedAt
	}
	row.StoreID = resolved.Store.ID
	row.SiteCode = defaultString(strings.TrimSpace(req.SiteCode), resolved.SiteCode)
	row.CaseType = normalizeSupportCaseType(req.CaseType)
	if row.CaseType == "" {
		row.CaseType = supportCaseTypeAfterSales
	}
	row.SourceType = defaultString(strings.TrimSpace(req.SourceType), supportSourceTypeManual)
	row.SourceRefType = strings.TrimSpace(req.SourceRefType)
	row.SourceRefID = req.SourceRefID
	row.OrderID = resolved.OrderID
	row.ReturnOrderID = resolved.ReturnOrderID
	row.ExternalCaseID = strings.TrimSpace(req.ExternalCaseID)
	row.Subject = defaultString(strings.TrimSpace(req.Subject), truncateSupportExcerpt(req.MessageBody))
	row.BuyerName = defaultString(strings.TrimSpace(req.BuyerName), resolved.BuyerName)
	row.BuyerEmail = defaultString(strings.TrimSpace(req.BuyerEmail), resolved.BuyerEmail)
	row.FirstReceivedAt = firstReceivedAt
	row.DueAt = timePtr(firstReceivedAt.UTC().Add(24 * time.Hour))
	if row.ReadStatus == "" {
		row.ReadStatus = supportReadStatusUnread
	}
	if row.HandlingStatus == "" {
		row.HandlingStatus = supportHandlingStatusPending
	}
	rawSource := decodeJSONMap(row.RawSourceJSON)
	for key, value := range req.RawSource {
		rawSource[key] = value
	}
	if strings.TrimSpace(req.Notes) != "" {
		rawSource["notes"] = strings.TrimSpace(req.Notes)
	}
	row.RawSourceJSON = encodeJSONObject(rawSource)
	var initialMessage *amazonModel.SupportCaseMessage
	if strings.TrimSpace(req.MessageBody) != "" {
		now := time.Now().UTC()
		row.LatestExcerpt = truncateSupportExcerpt(req.MessageBody)
		row.LastCustomerAt = &now
		role := supportMessageRoleCustomer
		channel := supportMessageChannelImported
		if !importMode {
			channel = supportMessageChannelInternal
			if row.ID == 0 {
				channel = supportMessageChannelManualCopy
			}
		}
		if !importMode && row.ID > 0 {
			role = supportMessageRoleInternal
		}
		message := amazonModel.SupportCaseMessage{
			Role:       role,
			Channel:    channel,
			BodyPlain:  strings.TrimSpace(req.MessageBody),
			SendStatus: supportSendStatusDraft,
			SentAt:     &now,
		}
		initialMessage = &message
	}
	if err := tx.Save(&row).Error; err != nil {
		return amazonModel.SupportCase{}, nil, err
	}
	if initialMessage != nil {
		initialMessage.CaseID = row.ID
		if err := tx.Create(initialMessage).Error; err != nil {
			return amazonModel.SupportCase{}, nil, err
		}
	}
	return row, initialMessage, nil
}

type supportResolvedAssociations struct {
	Store         amazonModel.StoreAccount
	SiteCode      string
	BuyerName     string
	BuyerEmail    string
	OrderID       *uint
	ReturnOrderID *uint
}

func (s *SupportInboxService) resolveSupportAssociationsTx(ctx context.Context, tx *gorm.DB, req amazonReq.AmazonSupportCaseUpsertReq) (supportResolvedAssociations, error) {
	resolved := supportResolvedAssociations{}
	var store amazonModel.StoreAccount
	if req.StoreID > 0 {
		if err := tx.WithContext(ctx).First(&store, req.StoreID).Error; err != nil {
			return resolved, err
		}
	}
	var order amazonModel.Order
	if req.OrderID != nil {
		if err := tx.WithContext(ctx).First(&order, *req.OrderID).Error; err != nil {
			return resolved, err
		}
		store = resolvedSupportStore(store, order.StoreID, tx.WithContext(ctx))
		resolved.OrderID = &order.ID
		resolved.SiteCode = order.SiteCode
		resolved.BuyerName = order.BuyerName
		resolved.BuyerEmail = order.BuyerEmail
	}
	var returnOrder amazonModel.ReturnOrder
	if req.ReturnOrderID != nil {
		if err := tx.WithContext(ctx).First(&returnOrder, *req.ReturnOrderID).Error; err != nil {
			return resolved, err
		}
		store = resolvedSupportStore(store, returnOrder.StoreID, tx.WithContext(ctx))
		resolved.ReturnOrderID = &returnOrder.ID
		if resolved.OrderID == nil && returnOrder.OrderID != nil {
			resolved.OrderID = returnOrder.OrderID
		}
		resolved.SiteCode = defaultString(resolved.SiteCode, returnOrder.SiteCode)
	}
	if store.ID == 0 {
		return resolved, errors.New("未找到关联店铺")
	}
	resolved.Store = store
	if resolved.SiteCode == "" {
		resolved.SiteCode = strings.TrimSpace(req.SiteCode)
	}
	return resolved, nil
}

func decodeSupportActionAvailability(raw datatypes.JSON) []SupportActionAvailability {
	source := decodeJSONMap(raw)
	items, _ := source["messagingActions"].([]interface{})
	result := make([]SupportActionAvailability, 0, len(items))
	for _, item := range items {
		mapped, _ := item.(map[string]interface{})
		if len(mapped) == 0 {
			continue
		}
		result = append(result, SupportActionAvailability{
			ActionKey:      strings.TrimSpace(fmt.Sprintf("%v", mapped["actionKey"])),
			Title:          strings.TrimSpace(fmt.Sprintf("%v", mapped["title"])),
			Path:           strings.TrimSpace(fmt.Sprintf("%v", mapped["path"])),
			SupportsText:   fmt.Sprintf("%v", mapped["supportsText"]) == "true",
			SupportsAttach: fmt.Sprintf("%v", mapped["supportsAttach"]) == "true",
		})
	}
	return result
}

func loadSupportReferenceMaps(ctx context.Context, rows []amazonModel.SupportCase) (map[uint]amazonModel.StoreAccount, map[uint]amazonModel.Order, map[uint]amazonModel.ReturnOrder, error) {
	storeIDs := make([]uint, 0, len(rows))
	orderIDs := make([]uint, 0, len(rows))
	returnIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		storeIDs = append(storeIDs, row.StoreID)
		orderIDs = append(orderIDs, valueOrZeroUint(row.OrderID))
		returnIDs = append(returnIDs, valueOrZeroUint(row.ReturnOrderID))
	}
	storeMap := map[uint]amazonModel.StoreAccount{}
	orderMap := map[uint]amazonModel.Order{}
	returnMap := map[uint]amazonModel.ReturnOrder{}
	if len(uniqueUintSlice(storeIDs)) > 0 {
		var stores []amazonModel.StoreAccount
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(storeIDs)).Find(&stores).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, store := range stores {
			storeMap[store.ID] = store
		}
	}
	if len(uniqueUintSlice(orderIDs)) > 0 {
		var orders []amazonModel.Order
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(orderIDs)).Find(&orders).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, order := range orders {
			orderMap[order.ID] = order
		}
	}
	if len(uniqueUintSlice(returnIDs)) > 0 {
		var returns []amazonModel.ReturnOrder
		if err := global.GVA_DB.WithContext(ctx).Where("id IN ?", uniqueUintSlice(returnIDs)).Find(&returns).Error; err != nil {
			return nil, nil, nil, err
		}
		for _, returnOrder := range returns {
			returnMap[returnOrder.ID] = returnOrder
		}
	}
	return storeMap, orderMap, returnMap, nil
}

func supportSLAMetrics(dueAt *time.Time, handlingStatus string, now time.Time) (string, int64) {
	if dueAt == nil {
		return supportSLABucketNormal, 0
	}
	remaining := int64(dueAt.Sub(now).Minutes())
	if handlingStatus == supportHandlingStatusClosed {
		return supportSLABucketNormal, remaining
	}
	if remaining < 0 {
		return supportSLABucketOverdue, remaining
	}
	if remaining <= int64(supportWarningLeadHours*60) {
		return supportSLABucketWarning, remaining
	}
	return supportSLABucketNormal, remaining
}

func supportWorkbookHeaderMap(headerRow []string) map[string]int {
	result := make(map[string]int, len(headerRow))
	for index, cell := range headerRow {
		key := strings.ToLower(strings.TrimSpace(cell))
		if key != "" {
			result[key] = index
		}
	}
	return result
}

func supportWorkbookCell(headerMap map[string]int, row []string, key string) string {
	index, ok := headerMap[key]
	if !ok || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func supportWorkbookRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func truncateSupportExcerpt(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 140 {
		return value
	}
	return string(runes[:140]) + "..."
}

func parseSupportTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}
	return nil
}

func normalizeSupportCaseType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case supportCaseTypeBuyerMessage:
		return supportCaseTypeBuyerMessage
	case supportCaseTypeReturn:
		return supportCaseTypeReturn
	case supportCaseTypeNegativeFeedback:
		return supportCaseTypeNegativeFeedback
	case supportCaseTypeAToZ, "atoz":
		return supportCaseTypeAToZ
	default:
		return supportCaseTypeAfterSales
	}
}

func normalizeSupportReadStatus(value string) string {
	if strings.ToLower(strings.TrimSpace(value)) == supportReadStatusRead {
		return supportReadStatusRead
	}
	return supportReadStatusUnread
}

func normalizeSupportHandlingStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case supportHandlingStatusProcessing:
		return supportHandlingStatusProcessing
	case supportHandlingStatusClosed:
		return supportHandlingStatusClosed
	default:
		return supportHandlingStatusPending
	}
}

func normalizeSupportDeliveryMode(value string) string {
	if strings.ToLower(strings.TrimSpace(value)) == supportDeliveryModeAmazonDirect {
		return supportDeliveryModeAmazonDirect
	}
	return supportDeliveryModeManualCopy
}

func renderSupportTemplate(template string, variables map[string]string) string {
	return strings.TrimSpace(supportTemplateVariablePattern.ReplaceAllStringFunc(template, func(match string) string {
		parts := supportTemplateVariablePattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		return variables[strings.TrimSpace(parts[1])]
	}))
}

func resolvedSupportStore(store amazonModel.StoreAccount, storeID uint, db *gorm.DB) amazonModel.StoreAccount {
	if store.ID > 0 || storeID == 0 {
		return store
	}
	_ = db.First(&store, storeID).Error
	return store
}
