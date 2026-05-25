package amazon

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/xuri/excelize/v2"
)

func TestSupportListSortsUnreadAndSLA(t *testing.T) {
	setupSupportTestDB(t)

	store := createSupportStore(t, 1, "Support Store")
	now := time.Now().UTC()

	createSupportCaseRecord(t, amazonModel.SupportCase{
		GVA_MODEL:       global.GVA_MODEL{ID: 1},
		StoreID:         store.ID,
		SiteCode:        "US",
		CaseType:        supportCaseTypeBuyerMessage,
		SourceType:      supportSourceTypeImport,
		Subject:         "已超时未读",
		ReadStatus:      supportReadStatusUnread,
		HandlingStatus:  supportHandlingStatusPending,
		FirstReceivedAt: timePtr(now.Add(-25 * time.Hour)),
		DueAt:           timePtr(now.Add(-1 * time.Hour)),
		LastCustomerAt:  timePtr(now.Add(-2 * time.Hour)),
	})
	createSupportCaseRecord(t, amazonModel.SupportCase{
		GVA_MODEL:       global.GVA_MODEL{ID: 2},
		StoreID:         store.ID,
		SiteCode:        "US",
		CaseType:        supportCaseTypeAfterSales,
		SourceType:      supportSourceTypeManual,
		Subject:         "预警未读",
		ReadStatus:      supportReadStatusUnread,
		HandlingStatus:  supportHandlingStatusPending,
		FirstReceivedAt: timePtr(now.Add(-22 * time.Hour)),
		DueAt:           timePtr(now.Add(2 * time.Hour)),
		LastCustomerAt:  timePtr(now.Add(-30 * time.Minute)),
	})
	createSupportCaseRecord(t, amazonModel.SupportCase{
		GVA_MODEL:       global.GVA_MODEL{ID: 3},
		StoreID:         store.ID,
		SiteCode:        "US",
		CaseType:        supportCaseTypeReturn,
		SourceType:      supportSourceTypeManual,
		Subject:         "正常已读",
		ReadStatus:      supportReadStatusRead,
		HandlingStatus:  supportHandlingStatusProcessing,
		FirstReceivedAt: timePtr(now.Add(-5 * time.Hour)),
		DueAt:           timePtr(now.Add(8 * time.Hour)),
		LastCustomerAt:  timePtr(now.Add(-10 * time.Minute)),
	})

	result, err := new(SupportInboxService).List(context.Background(), amazonReq.AmazonSupportCaseListReq{
		PageInfo: commonReq.PageInfo{
			Page:     1,
			PageSize: 10,
		},
	})
	if err != nil {
		t.Fatalf("list support cases: %v", err)
	}

	if len(result.List) != 3 {
		t.Fatalf("expected 3 support cases, got %d", len(result.List))
	}
	if result.List[0].ID != 1 || result.List[1].ID != 2 || result.List[2].ID != 3 {
		t.Fatalf("unexpected list order: %+v", []uint{result.List[0].ID, result.List[1].ID, result.List[2].ID})
	}
	if result.Summary.AllCount != 3 || result.Summary.UnreadCount != 2 {
		t.Fatalf("unexpected summary counts: %+v", result.Summary)
	}
	if result.Summary.OverdueCount != 1 || result.Summary.WarningCount != 1 {
		t.Fatalf("unexpected sla summary: %+v", result.Summary)
	}
}

func TestSupportBuildImportUpsertReqAssociatesOrderAndReturn(t *testing.T) {
	setupSupportTestDB(t)

	store := createSupportStore(t, 1, "Import Store")
	order := createSupportOrder(t, 10, store.ID, "AMZ-ORDER-1", "US")
	returnOrder := createSupportReturnOrder(t, 20, store.ID, order.ID, "AMZ-RMA-1", "US")

	headers := supportWorkbookHeaderMap([]string{
		"store_name", "site_code", "case_type", "subject", "amazon_order_id", "amazon_rma_id",
		"buyer_name", "buyer_email", "received_at", "message_body", "external_case_id", "notes",
	})
	row := []string{
		store.StoreName,
		"",
		supportCaseTypeReturn,
		"退货说明",
		order.AmazonOrderID,
		returnOrder.AmazonRMAID,
		"Buyer",
		"buyer@example.com",
		"2026-01-02 03:04:05",
		"请协助处理退货",
		"EXT-001",
		"需要优先处理",
	}

	req, err := new(SupportInboxService).buildSupportImportUpsertReq(context.Background(), headers, row)
	if err != nil {
		t.Fatalf("build import upsert req: %v", err)
	}

	if req.StoreID != store.ID {
		t.Fatalf("expected store id %d, got %d", store.ID, req.StoreID)
	}
	if req.OrderID == nil || *req.OrderID != order.ID {
		t.Fatalf("expected order association %d, got %+v", order.ID, req.OrderID)
	}
	if req.ReturnOrderID == nil || *req.ReturnOrderID != returnOrder.ID {
		t.Fatalf("expected return association %d, got %+v", returnOrder.ID, req.ReturnOrderID)
	}
	if req.SourceType != supportSourceTypeImport || req.SourceRefType != "workbook" {
		t.Fatalf("unexpected import source: %s %s", req.SourceType, req.SourceRefType)
	}
	if req.SiteCode != "US" || req.ExternalCaseID != "EXT-001" {
		t.Fatalf("unexpected import req payload: %+v", req)
	}
}

func TestSupportImportWorkbookPartialSuccess(t *testing.T) {
	setupSupportTestDB(t)

	store := createSupportStore(t, 1, "Workbook Store")
	order := createSupportOrder(t, 10, store.ID, "AMZ-ORDER-2", "CA")
	createSupportReturnOrder(t, 20, store.ID, order.ID, "AMZ-RMA-2", "CA")

	workbook := excelize.NewFile()
	sheet := workbook.GetSheetName(0)
	_ = workbook.SetSheetRow(sheet, "A1", &[]interface{}{
		"store_name", "site_code", "case_type", "subject", "amazon_order_id", "amazon_rma_id",
		"buyer_name", "buyer_email", "received_at", "message_body", "external_case_id", "notes",
	})
	_ = workbook.SetSheetRow(sheet, "A2", &[]interface{}{
		store.StoreName, "", supportCaseTypeBuyerMessage, "Buyer message", order.AmazonOrderID, "AMZ-RMA-2",
		"Buyer", "buyer@example.com", "2026-02-03 04:05:06", "买家来信内容", "EXT-OK", "备注1",
	})
	_ = workbook.SetSheetRow(sheet, "A3", &[]interface{}{
		"Missing Store", "", supportCaseTypeBuyerMessage, "Invalid row", "", "",
		"", "", "2026-02-03 04:05:06", "无效行", "EXT-BAD", "",
	})
	buffer, err := workbook.WriteToBuffer()
	if err != nil {
		t.Fatalf("write workbook buffer: %v", err)
	}

	result, err := new(SupportInboxService).ImportWorkbook(context.Background(), "support.xlsx", buffer.Bytes())
	if err != nil {
		t.Fatalf("import workbook: %v", err)
	}

	if result.TotalRows != 2 || result.SuccessRows != 1 || result.FailedRows != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 import error, got %+v", result.Errors)
	}

	var caseCount int64
	if err := global.GVA_DB.Model(&amazonModel.SupportCase{}).Count(&caseCount).Error; err != nil {
		t.Fatalf("count support cases: %v", err)
	}
	if caseCount != 1 {
		t.Fatalf("expected 1 imported support case, got %d", caseCount)
	}

	var message amazonModel.SupportCaseMessage
	if err := global.GVA_DB.First(&message).Error; err != nil {
		t.Fatalf("load imported message: %v", err)
	}
	if message.Channel != supportMessageChannelImported || message.Role != supportMessageRoleCustomer {
		t.Fatalf("unexpected imported message payload: %+v", message)
	}
}

func TestSupportStatusTransitionsAndManualReply(t *testing.T) {
	setupSupportTestDB(t)

	store := createSupportStore(t, 1, "Reply Store")
	service := new(SupportInboxService)
	templateService := new(SupportTemplateService)

	detail, err := service.UpsertCase(context.Background(), amazonReq.AmazonSupportCaseUpsertReq{
		StoreID:         store.ID,
		SiteCode:        "US",
		CaseType:        supportCaseTypeAfterSales,
		SourceType:      supportSourceTypeManual,
		SourceRefType:   "order",
		ExternalCaseID:  "CASE-001",
		Subject:         "订单售后跟进",
		BuyerName:       "Alice",
		BuyerEmail:      "alice@example.com",
		FirstReceivedAt: "2026-03-04 05:06:07",
		MessageBody:     "客户需要售后帮助",
	})
	if err != nil {
		t.Fatalf("create support case: %v", err)
	}

	if _, err := service.MarkRead(context.Background(), detail.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	pendingDetail, err := service.MarkPending(context.Background(), detail.ID)
	if err != nil {
		t.Fatalf("mark pending: %v", err)
	}
	if pendingDetail.HandlingStatus != supportHandlingStatusPending || pendingDetail.ReadStatus != supportReadStatusRead {
		t.Fatalf("unexpected pending detail: %+v", pendingDetail)
	}

	templatePage, err := templateService.List(context.Background(), amazonReq.AmazonSupportTemplateListReq{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("list support templates: %v", err)
	}
	var templateID uint
	for _, item := range templatePage.List {
		if item.Code == "after_sales_followup" {
			templateID = item.ID
			break
		}
	}
	if templateID == 0 {
		t.Fatalf("after_sales_followup template not found")
	}

	replyResult, err := service.SendReply(context.Background(), amazonReq.AmazonSupportSendReplyReq{
		CaseID:       detail.ID,
		TemplateID:   templateID,
		DeliveryMode: supportDeliveryModeManualCopy,
		Variables: map[string]string{
			"resolution_note": "我们已经安排补发。",
		},
	})
	if err != nil {
		t.Fatalf("send support reply: %v", err)
	}

	if replyResult.SendStatus != supportSendStatusCopied {
		t.Fatalf("unexpected reply result: %+v", replyResult)
	}
	if !bytes.Contains([]byte(replyResult.RenderedBody), []byte("我们已经安排补发。")) {
		t.Fatalf("rendered body missing resolution note: %s", replyResult.RenderedBody)
	}

	var row amazonModel.SupportCase
	if err := global.GVA_DB.First(&row, detail.ID).Error; err != nil {
		t.Fatalf("load support case after reply: %v", err)
	}
	if row.HandlingStatus != supportHandlingStatusProcessing || row.ReadStatus != supportReadStatusRead || row.LastAgentAt == nil {
		t.Fatalf("unexpected support case after reply: %+v", row)
	}

	var messages []amazonModel.SupportCaseMessage
	if err := global.GVA_DB.Where("case_id = ?", detail.ID).Order("id ASC").Find(&messages).Error; err != nil {
		t.Fatalf("load support messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 timeline messages, got %d", len(messages))
	}
	last := messages[len(messages)-1]
	if last.SendStatus != supportSendStatusCopied || last.Channel != supportMessageChannelManualCopy || last.TemplateKey != "after_sales_followup" {
		t.Fatalf("unexpected reply timeline message: %+v", last)
	}

	closedDetail, err := service.Close(context.Background(), detail.ID)
	if err != nil {
		t.Fatalf("close support case: %v", err)
	}
	if closedDetail.HandlingStatus != supportHandlingStatusClosed {
		t.Fatalf("unexpected closed detail: %+v", closedDetail)
	}
}

func TestExtractSupportMessagingActions(t *testing.T) {
	resp := map[string]interface{}{
		"payload": map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{
					"name": "createNegativeFeedbackRemoval",
					"href": "/messaging/v1/orders/123/messages/createNegativeFeedbackRemoval?marketplaceIds=ATVPDKIKX0DER",
					"schema": map[string]interface{}{
						"properties": map[string]interface{}{
							"text": map[string]interface{}{},
						},
					},
				},
			},
		},
	}

	actions := extractSupportMessagingActions(resp)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %+v", actions)
	}
	if actions[0].ActionKey != "createNegativeFeedbackRemoval" || !actions[0].SupportsText {
		t.Fatalf("unexpected action payload: %+v", actions[0])
	}
}

func setupSupportTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "support.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&amazonModel.StoreAccount{},
		&amazonModel.Order{},
		&amazonModel.ReturnOrder{},
		&amazonModel.SupportCase{},
		&amazonModel.SupportCaseMessage{},
		&amazonModel.SupportTemplate{},
		&amazonModel.SupportImportJob{},
	); err != nil {
		t.Fatalf("migrate support tables: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
}

func createSupportStore(t *testing.T, id uint, name string) amazonModel.StoreAccount {
	t.Helper()

	store := amazonModel.StoreAccount{
		GVA_MODEL:  global.GVA_MODEL{ID: id},
		StoreName:  name,
		Region:     "NA",
		AuthStatus: "authorized",
		IsEnabled:  true,
	}
	if err := global.GVA_DB.Create(&store).Error; err != nil {
		t.Fatalf("create support store: %v", err)
	}
	return store
}

func createSupportOrder(t *testing.T, id, storeID uint, amazonOrderID, siteCode string) amazonModel.Order {
	t.Helper()

	order := amazonModel.Order{
		GVA_MODEL:       global.GVA_MODEL{ID: id},
		StoreID:         storeID,
		AmazonOrderID:   amazonOrderID,
		SiteCode:        siteCode,
		MarketplaceID:   "ATVPDKIKX0DER",
		OrderStatus:     "Unshipped",
		FulfillmentType: "fbm",
		BuyerName:       "Test Buyer",
		BuyerEmail:      "buyer@example.com",
	}
	if err := global.GVA_DB.Create(&order).Error; err != nil {
		t.Fatalf("create support order: %v", err)
	}
	return order
}

func createSupportReturnOrder(t *testing.T, id, storeID, orderID uint, amazonRMAID, siteCode string) amazonModel.ReturnOrder {
	t.Helper()

	orderIDCopy := orderID
	returnOrder := amazonModel.ReturnOrder{
		GVA_MODEL:           global.GVA_MODEL{ID: id},
		StoreID:             storeID,
		OrderID:             &orderIDCopy,
		AmazonOrderID:       "AMZ-ORDER-" + amazonRMAID,
		SiteCode:            siteCode,
		AmazonRMAID:         amazonRMAID,
		ReturnType:          "BuyerRemorse",
		Resolution:          "Refund",
		TrackingID:          "TRACK-" + amazonRMAID,
		ReturnRequestStatus: "Pending",
	}
	if err := global.GVA_DB.Create(&returnOrder).Error; err != nil {
		t.Fatalf("create support return order: %v", err)
	}
	return returnOrder
}

func createSupportCaseRecord(t *testing.T, row amazonModel.SupportCase) {
	t.Helper()
	if err := global.GVA_DB.Create(&row).Error; err != nil {
		t.Fatalf("create support case record: %v", err)
	}
}
