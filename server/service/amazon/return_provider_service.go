package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
)

type ReturnProviderService struct{}

type ReturnProviderClient interface {
	Quote(ctx context.Context, req ReturnProviderQuoteRequest) (ReturnProviderQuoteResult, error)
	CreateDisposition(ctx context.Context, req ReturnDispositionCreateRequest) (ReturnDispositionCreateResult, error)
	QueryDisposition(ctx context.Context, providerOrderNo string) (ReturnDispositionTrackingResult, error)
}

type ReturnProviderQuoteRequest struct {
	Provider         amazonModel.ReturnServiceProvider
	ReturnOrder      amazonModel.ReturnOrder
	ReturnItem       amazonModel.ReturnItem
	WeightKG         float64
	CountryCode      string
	DestinationType  string
	SupportsRedirect bool
}

type ReturnProviderQuoteResult struct {
	QuoteFeeCNY    float64 `json:"quoteFeeCny"`
	HandlingFeeCNY float64 `json:"handlingFeeCny"`
	TotalFeeCNY    float64 `json:"totalFeeCny"`
}

type ReturnDispositionCreateRequest struct {
	Provider    amazonModel.ReturnServiceProvider `json:"provider"`
	ReturnOrder amazonModel.ReturnOrder           `json:"returnOrder"`
	ReturnItem  amazonModel.ReturnItem            `json:"returnItem"`
	TargetType  string                            `json:"targetType"`
	Destination map[string]interface{}            `json:"destination"`
}

type ReturnDispositionCreateResult struct {
	ProviderOrderNo    string `json:"providerOrderNo"`
	ProviderTrackingNo string `json:"providerTrackingNo"`
	LabelURL           string `json:"labelUrl"`
}

type ReturnDispositionTrackingResult struct {
	Status      string `json:"status"`
	CompletedAt string `json:"completedAt"`
}

var returnProviderMu sync.RWMutex
var returnProviderOverrides = map[string]ReturnProviderClient{}

func registerReturnProviderOverride(code string, client ReturnProviderClient) {
	returnProviderMu.Lock()
	defer returnProviderMu.Unlock()
	if client == nil {
		delete(returnProviderOverrides, strings.TrimSpace(strings.ToLower(code)))
		return
	}
	returnProviderOverrides[strings.TrimSpace(strings.ToLower(code))] = client
}

func (s *ReturnProviderService) List(ctx context.Context, req amazonReq.ReturnProviderListReq) (ReturnServiceProviderPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnServiceProvider{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("name LIKE ? OR code LIKE ?", keyword, keyword)
	}
	if strings.TrimSpace(req.QuoteMode) != "" {
		db = db.Where("quote_mode = ?", strings.TrimSpace(req.QuoteMode))
	}
	if req.IsEnabled != nil {
		db = db.Where("is_enabled = ?", *req.IsEnabled)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ReturnServiceProviderPageResult{}, err
	}
	var rows []amazonModel.ReturnServiceProvider
	if err := db.Scopes(req.PageInfo.Paginate()).Order("priority ASC, id DESC").Find(&rows).Error; err != nil {
		return ReturnServiceProviderPageResult{}, err
	}
	result := ReturnServiceProviderPageResult{
		List:     make([]ReturnServiceProviderDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, buildReturnProviderDetail(row))
	}
	return result, nil
}

func (s *ReturnProviderService) Find(ctx context.Context, id uint) (ReturnServiceProviderDetail, error) {
	if id == 0 {
		return ReturnServiceProviderDetail{}, errors.New("id is required")
	}
	var row amazonModel.ReturnServiceProvider
	if err := global.GVA_DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return ReturnServiceProviderDetail{}, err
	}
	return buildReturnProviderDetail(row), nil
}

func (s *ReturnProviderService) Upsert(ctx context.Context, req amazonReq.ReturnProviderUpsertReq) (ReturnServiceProviderDetail, error) {
	if strings.TrimSpace(req.Name) == "" {
		return ReturnServiceProviderDetail{}, errors.New("name is required")
	}
	if strings.TrimSpace(req.Code) == "" {
		return ReturnServiceProviderDetail{}, errors.New("code is required")
	}
	var row amazonModel.ReturnServiceProvider
	db := global.GVA_DB.WithContext(ctx)
	if req.ID > 0 {
		if err := db.First(&row, req.ID).Error; err != nil {
			return ReturnServiceProviderDetail{}, err
		}
	}
	row.Name = strings.TrimSpace(req.Name)
	row.Code = strings.TrimSpace(strings.ToLower(req.Code))
	row.QuoteMode = defaultString(strings.TrimSpace(req.QuoteMode), returnQuoteModeManual)
	row.BaseURL = strings.TrimSpace(req.BaseURL)
	row.QuotePath = strings.TrimSpace(req.QuotePath)
	row.CreatePath = strings.TrimSpace(req.CreatePath)
	row.TrackingPath = strings.TrimSpace(req.TrackingPath)
	row.AuthHeader = strings.TrimSpace(req.AuthHeader)
	row.HandlingFeeCNY = cloneFloat64(req.HandlingFeeCNY)
	row.BaseFeeCNY = cloneFloat64(req.BaseFeeCNY)
	row.PerKGFeeCNY = cloneFloat64(req.PerKGFeeCNY)
	row.SupportsBuyerRedirect = req.SupportsBuyerRedirect
	row.SupportsWarehouseReturn = req.SupportsWarehouseReturn
	row.SupportsTracking = req.SupportsTracking
	row.SupportsAddressPrefill = req.SupportsAddressPrefill
	row.CountryScopesJSON = encodeJSON(uniqueStrings(req.CountryScopes))
	row.Priority = req.Priority
	row.IsEnabled = req.IsEnabled
	if strings.TrimSpace(req.AuthToken) != "" {
		encrypted, err := encryptAmazonSecret(strings.TrimSpace(req.AuthToken))
		if err != nil {
			return ReturnServiceProviderDetail{}, err
		}
		row.AuthTokenEncrypted = encrypted
	}
	if err := db.Save(&row).Error; err != nil {
		return ReturnServiceProviderDetail{}, err
	}
	return buildReturnProviderDetail(row), nil
}

func (s *ReturnProviderService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("id is required")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&amazonModel.ReturnServiceProvider{}, id).Error
}

func (s *ReturnProviderService) TestConnection(ctx context.Context, id uint) (ReturnTestConnectionResult, error) {
	if id == 0 {
		return ReturnTestConnectionResult{}, errors.New("id is required")
	}
	var row amazonModel.ReturnServiceProvider
	if err := global.GVA_DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return ReturnTestConnectionResult{}, err
	}
	client, err := resolveReturnProviderClient(row)
	if err != nil {
		return ReturnTestConnectionResult{}, err
	}
	if row.QuoteMode == returnQuoteModeManual {
		return ReturnTestConnectionResult{ID: id, Reachable: true, Message: "manual provider ready"}, nil
	}
	configured, ok := client.(*configuredReturnProviderClient)
	if !ok {
		return ReturnTestConnectionResult{ID: id, Reachable: true, Message: "override provider ready"}, nil
	}
	if strings.TrimSpace(row.BaseURL) == "" {
		return ReturnTestConnectionResult{}, errors.New("baseUrl is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(row.BaseURL, "/"), nil)
	if err != nil {
		return ReturnTestConnectionResult{}, err
	}
	if configured.authHeader != "" && configured.authToken != "" {
		req.Header.Set(configured.authHeader, configured.authToken)
	}
	resp, err := configured.httpClient.Do(req)
	if err != nil {
		_ = global.GVA_DB.WithContext(ctx).Model(&row).Update("last_error", err.Error()).Error
		return ReturnTestConnectionResult{}, err
	}
	_ = resp.Body.Close()
	message := fmt.Sprintf("provider returned %d", resp.StatusCode)
	_ = global.GVA_DB.WithContext(ctx).Model(&row).Update("last_error", "").Error
	return ReturnTestConnectionResult{ID: id, Reachable: resp.StatusCode < 500, Message: message}, nil
}

func buildReturnProviderDetail(row amazonModel.ReturnServiceProvider) ReturnServiceProviderDetail {
	return ReturnServiceProviderDetail{
		ID:                      row.ID,
		Name:                    row.Name,
		Code:                    row.Code,
		QuoteMode:               row.QuoteMode,
		BaseURL:                 row.BaseURL,
		QuotePath:               row.QuotePath,
		CreatePath:              row.CreatePath,
		TrackingPath:            row.TrackingPath,
		AuthHeader:              row.AuthHeader,
		HandlingFeeCNY:          cloneFloat64(row.HandlingFeeCNY),
		BaseFeeCNY:              cloneFloat64(row.BaseFeeCNY),
		PerKGFeeCNY:             cloneFloat64(row.PerKGFeeCNY),
		SupportsBuyerRedirect:   row.SupportsBuyerRedirect,
		SupportsWarehouseReturn: row.SupportsWarehouseReturn,
		SupportsTracking:        row.SupportsTracking,
		SupportsAddressPrefill:  row.SupportsAddressPrefill,
		CountryScopes:           decodeStringJSON(row.CountryScopesJSON),
		Priority:                row.Priority,
		IsEnabled:               row.IsEnabled,
		LastError:               row.LastError,
	}
}

type configuredReturnProviderClient struct {
	provider   amazonModel.ReturnServiceProvider
	httpClient *http.Client
	authHeader string
	authToken  string
}

func resolveReturnProviderClient(provider amazonModel.ReturnServiceProvider) (ReturnProviderClient, error) {
	code := strings.TrimSpace(strings.ToLower(provider.Code))
	returnProviderMu.RLock()
	override := returnProviderOverrides[code]
	returnProviderMu.RUnlock()
	if override != nil {
		return override, nil
	}
	authToken := ""
	if strings.TrimSpace(provider.AuthTokenEncrypted) != "" {
		decrypted, err := decryptAmazonSecret(provider.AuthTokenEncrypted)
		if err != nil {
			return nil, err
		}
		authToken = decrypted
	}
	return &configuredReturnProviderClient{
		provider:   provider,
		httpClient: &http.Client{},
		authHeader: provider.AuthHeader,
		authToken:  authToken,
	}, nil
}

func (c *configuredReturnProviderClient) Quote(ctx context.Context, req ReturnProviderQuoteRequest) (ReturnProviderQuoteResult, error) {
	handling := floatOrZero(c.provider.HandlingFeeCNY)
	if c.provider.QuoteMode != returnQuoteModeAPI {
		quote := floatOrZero(c.provider.BaseFeeCNY) + req.WeightKG*floatOrZero(c.provider.PerKGFeeCNY)
		return ReturnProviderQuoteResult{
			QuoteFeeCNY:    quote,
			HandlingFeeCNY: handling,
			TotalFeeCNY:    quote + handling,
		}, nil
	}
	response, err := c.doJSON(ctx, http.MethodPost, c.provider.QuotePath, map[string]interface{}{
		"countryCode":     req.CountryCode,
		"destinationType": req.DestinationType,
		"weightKg":        req.WeightKG,
		"sellerSku":       req.ReturnItem.SellerSKU,
		"returnQuantity":  req.ReturnItem.ReturnQuantity,
	})
	if err != nil {
		return ReturnProviderQuoteResult{}, err
	}
	quote := toFloat(response["quoteFeeCny"])
	handlingFromAPI := toFloat(response["handlingFeeCny"])
	if handlingFromAPI == 0 {
		handlingFromAPI = handling
	}
	return ReturnProviderQuoteResult{
		QuoteFeeCNY:    quote,
		HandlingFeeCNY: handlingFromAPI,
		TotalFeeCNY:    quote + handlingFromAPI,
	}, nil
}

func (c *configuredReturnProviderClient) CreateDisposition(ctx context.Context, req ReturnDispositionCreateRequest) (ReturnDispositionCreateResult, error) {
	if c.provider.QuoteMode != returnQuoteModeAPI {
		return ReturnDispositionCreateResult{
			ProviderOrderNo:    fmt.Sprintf("%s-%d-%d", strings.ToUpper(c.provider.Code), req.ReturnOrder.ID, req.ReturnItem.ID),
			ProviderTrackingNo: fmt.Sprintf("RTN-%d", req.ReturnItem.ID),
		}, nil
	}
	response, err := c.doJSON(ctx, http.MethodPost, c.provider.CreatePath, map[string]interface{}{
		"amazonRmaId":   req.ReturnOrder.AmazonRMAID,
		"amazonOrderId": req.ReturnOrder.AmazonOrderID,
		"targetType":    req.TargetType,
		"sellerSku":     req.ReturnItem.SellerSKU,
		"quantity":      req.ReturnItem.ReturnQuantity,
		"destination":   req.Destination,
	})
	if err != nil {
		return ReturnDispositionCreateResult{}, err
	}
	return ReturnDispositionCreateResult{
		ProviderOrderNo:    strings.TrimSpace(fmt.Sprintf("%v", response["providerOrderNo"])),
		ProviderTrackingNo: strings.TrimSpace(fmt.Sprintf("%v", response["providerTrackingNo"])),
		LabelURL:           strings.TrimSpace(fmt.Sprintf("%v", response["labelUrl"])),
	}, nil
}

func (c *configuredReturnProviderClient) QueryDisposition(ctx context.Context, providerOrderNo string) (ReturnDispositionTrackingResult, error) {
	if c.provider.QuoteMode != returnQuoteModeAPI || strings.TrimSpace(c.provider.TrackingPath) == "" {
		return ReturnDispositionTrackingResult{}, nil
	}
	query := url.Values{}
	query.Set("providerOrderNo", providerOrderNo)
	response, err := c.doJSON(ctx, http.MethodGet, c.provider.TrackingPath+"?"+query.Encode(), nil)
	if err != nil {
		return ReturnDispositionTrackingResult{}, err
	}
	return ReturnDispositionTrackingResult{
		Status:      strings.TrimSpace(fmt.Sprintf("%v", response["status"])),
		CompletedAt: strings.TrimSpace(fmt.Sprintf("%v", response["completedAt"])),
	}, nil
}

func (c *configuredReturnProviderClient) doJSON(ctx context.Context, method, path string, payload map[string]interface{}) (map[string]interface{}, error) {
	if strings.TrimSpace(c.provider.BaseURL) == "" {
		return nil, errors.New("return provider baseUrl is required")
	}
	fullURL := strings.TrimRight(c.provider.BaseURL, "/")
	if strings.TrimSpace(path) != "" {
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			fullURL = path
		} else {
			fullURL += "/" + strings.TrimLeft(path, "/")
		}
	}
	var bodyReader *strings.Reader
	if payload == nil {
		bodyReader = strings.NewReader("")
	} else {
		raw := encodeJSONObject(payload)
		bodyReader = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.authHeader != "" && c.authToken != "" {
		req.Header.Set(c.authHeader, c.authToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var parsed map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("provider request failed: %d", resp.StatusCode)
		}
		return map[string]interface{}{}, nil
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider request failed: %d", resp.StatusCode)
	}
	return parsed, nil
}

func toFloat(value interface{}) float64 {
	parsed, ok := parseNumericValue(value)
	if ok {
		return parsed
	}
	return 0
}
