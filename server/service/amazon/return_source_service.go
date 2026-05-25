package amazon

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

const (
	amazonReturnReportByReturnDate = "GET_FLAT_FILE_RETURNS_DATA_BY_RETURN_DATE"
	amazonReturnReportPrime        = "GET_CSV_MFN_PRIME_RETURNS_REPORT"
	amazonReportStatusDone         = "DONE"
	amazonReportStatusCancelled    = "CANCELLED"
	amazonReportStatusFatal        = "FATAL"
)

type ReturnReportRow struct {
	AmazonOrderID       string
	AmazonRMAID         string
	MerchantRMAID       string
	SellerSKU           string
	ASIN                string
	Title               string
	ReturnQuantity      int
	ReturnRequestDate   *time.Time
	ReturnRequestStatus string
	ReturnDeliveryDate  *time.Time
	ReturnType          string
	Resolution          string
	LabelCost           *float64
	LabelCurrency       string
	RefundAmount        *float64
	RefundCurrency      string
	Carrier             string
	TrackingID          string
	Raw                 map[string]interface{}
}

type ReturnSourceAdapter interface {
	RequestReport(ctx context.Context, store amazonModel.StoreAccount, reportType string, startAt, endAt time.Time) (string, error)
	PollReport(ctx context.Context, store amazonModel.StoreAccount, reportID string) (string, string, error)
	DownloadDocument(ctx context.Context, store amazonModel.StoreAccount, documentID string) ([]byte, error)
	ParseRows(reportType string, raw []byte) ([]ReturnReportRow, error)
}

var returnSourceAdapter ReturnSourceAdapter = &amazonReturnSourceAdapter{}

type amazonReturnSourceAdapter struct{}

func setReturnSourceAdapter(adapter ReturnSourceAdapter) {
	if adapter == nil {
		returnSourceAdapter = &amazonReturnSourceAdapter{}
		return
	}
	returnSourceAdapter = adapter
}

func (a *amazonReturnSourceAdapter) RequestReport(ctx context.Context, store amazonModel.StoreAccount, reportType string, startAt, endAt time.Time) (string, error) {
	payload := map[string]interface{}{
		"reportType":    reportType,
		"dataStartTime": startAt.UTC().Format(time.RFC3339),
		"dataEndTime":   endAt.UTC().Format(time.RFC3339),
	}
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodPost, "/reports/2021-06-30/reports", nil, payload, nil)
	if err != nil {
		return "", err
	}
	reportID := strings.TrimSpace(fmt.Sprintf("%v", resp["reportId"]))
	if reportID == "" {
		reportID = strings.TrimSpace(fmt.Sprintf("%v", resp["reportIdList"]))
	}
	if reportID == "" {
		payloadMap := extractPayloadMap(resp)
		reportID = strings.TrimSpace(fmt.Sprintf("%v", payloadMap["reportId"]))
	}
	if reportID == "" {
		return "", fmt.Errorf("Amazon 未返回 reportId: %s", reportType)
	}
	return reportID, nil
}

func (a *amazonReturnSourceAdapter) PollReport(ctx context.Context, store amazonModel.StoreAccount, reportID string) (string, string, error) {
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodGet, "/reports/2021-06-30/reports/"+reportID, nil, nil, nil)
	if err != nil {
		return "", "", err
	}
	payload := extractPayloadMap(resp)
	status := strings.TrimSpace(fmt.Sprintf("%v", payload["processingStatus"]))
	documentID := strings.TrimSpace(fmt.Sprintf("%v", payload["reportDocumentId"]))
	return status, documentID, nil
}

func (a *amazonReturnSourceAdapter) DownloadDocument(ctx context.Context, store amazonModel.StoreAccount, documentID string) ([]byte, error) {
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodGet, "/reports/2021-06-30/documents/"+documentID, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	payload := extractPayloadMap(resp)
	downloadURL := strings.TrimSpace(fmt.Sprintf("%v", payload["url"]))
	if downloadURL == "" {
		return nil, fmt.Errorf("Amazon 文档缺少下载地址: %s", documentID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	compression := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", payload["compressionAlgorithm"])))
	if compression == "GZIP" {
		reader, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	}
	return raw, nil
}

func (a *amazonReturnSourceAdapter) ParseRows(reportType string, raw []byte) ([]ReturnReportRow, error) {
	delimiter := ','
	if bytes.Contains(raw, []byte("\t")) {
		delimiter = '\t'
	}
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) <= 1 {
		return nil, nil
	}
	headers := make([]string, 0, len(records[0]))
	for _, header := range records[0] {
		headers = append(headers, normalizeReturnHeader(header))
	}
	rows := make([]ReturnReportRow, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) == 0 {
			continue
		}
		rowMap := map[string]string{}
		rawMap := map[string]interface{}{}
		for idx, header := range headers {
			if header == "" || idx >= len(record) {
				continue
			}
			value := strings.TrimSpace(record[idx])
			rowMap[header] = value
			rawMap[header] = value
		}
		amazonOrderID := firstNonEmpty(rowMap["orderid"], rowMap["amazonorderid"])
		amazonRMAID := firstNonEmpty(rowMap["amazonrmaid"], rowMap["rmaid"], rowMap["returnmerchandiseauthorizationid"])
		if amazonOrderID == "" || amazonRMAID == "" {
			continue
		}
		returnQuantity := parseIntDefault(rowMap["quantity"], 1)
		labelCost, _ := parseNumericValue(rowMap["labelcost"])
		refundAmount, _ := parseNumericValue(firstNonEmpty(rowMap["refundamount"], rowMap["refundamountvalue"]))
		row := ReturnReportRow{
			AmazonOrderID:       amazonOrderID,
			AmazonRMAID:         amazonRMAID,
			MerchantRMAID:       firstNonEmpty(rowMap["merchantrmaid"], rowMap["merchantauthorizationid"]),
			SellerSKU:           firstNonEmpty(rowMap["merchantsku"], rowMap["sku"]),
			ASIN:                rowMap["asin"],
			Title:               firstNonEmpty(rowMap["itemname"], rowMap["productname"], rowMap["title"]),
			ReturnQuantity:      returnQuantity,
			ReturnRequestDate:   parseAnyTime(firstNonEmpty(rowMap["returnrequestdate"], rowMap["requestdate"])),
			ReturnRequestStatus: firstNonEmpty(rowMap["returnrequeststatus"], rowMap["requeststatus"], rowMap["status"]),
			ReturnDeliveryDate:  parseAnyTime(firstNonEmpty(rowMap["returndeliverydate"], rowMap["deliverydate"])),
			ReturnType:          firstNonEmpty(rowMap["returntype"], rowMap["type"]),
			Resolution:          rowMap["resolution"],
			LabelCost:           optionalFloatPtr(labelCost, rowMap["labelcost"]),
			LabelCurrency:       firstNonEmpty(rowMap["labelcurrencycode"], rowMap["labelcostcurrencycode"], rowMap["currencycode"]),
			RefundAmount:        optionalFloatPtr(refundAmount, firstNonEmpty(rowMap["refundamount"], rowMap["refundamountvalue"])),
			RefundCurrency:      firstNonEmpty(rowMap["refundcurrencycode"], rowMap["currencycode"]),
			Carrier:             rowMap["carrier"],
			TrackingID:          firstNonEmpty(rowMap["trackingid"], rowMap["trackingnumber"]),
			Raw:                 rawMap,
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func normalizeReturnHeader(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer("-", "", "_", "", " ", "", "/", "", ".", "", "(", "", ")", "")
	return replacer.Replace(value)
}

func buildReturnSourceLineHash(row ReturnReportRow) string {
	payload := strings.Join([]string{
		row.AmazonOrderID,
		row.AmazonRMAID,
		row.SellerSKU,
		row.ASIN,
		fmt.Sprintf("%d", row.ReturnQuantity),
		defaultString(formatCollectorTime(row.ReturnRequestDate), ""),
		row.TrackingID,
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func optionalFloatPtr(parsed float64, raw string) *float64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	value := parsed
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseIntDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
