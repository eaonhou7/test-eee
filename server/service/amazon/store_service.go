package amazon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type StoreService struct{}

func (s *StoreService) List(ctx context.Context, req amazonReq.StoreAccountListReq) (StoreAccountPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.StoreAccount{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("store_name LIKE ? OR seller_id LIKE ? OR selling_partner_id LIKE ?", keyword, keyword, keyword)
	}
	if strings.TrimSpace(req.AuthStatus) != "" {
		db = db.Where("auth_status = ?", strings.TrimSpace(req.AuthStatus))
	}
	if req.IsEnabled != nil {
		db = db.Where("is_enabled = ?", *req.IsEnabled)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return StoreAccountPageResult{}, err
	}
	var stores []amazonModel.StoreAccount
	if err := db.Scopes(req.PageInfo.Paginate()).Order("id DESC").Find(&stores).Error; err != nil {
		return StoreAccountPageResult{}, err
	}
	result := StoreAccountPageResult{
		List:     make([]StoreAccountDetail, 0, len(stores)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, store := range stores {
		result.List = append(result.List, buildStoreAccountDetail(store))
	}
	return result, nil
}

func (s *StoreService) Find(ctx context.Context, id uint) (StoreAccountDetail, error) {
	if id == 0 {
		return StoreAccountDetail{}, errors.New("id is required")
	}
	var store amazonModel.StoreAccount
	if err := global.GVA_DB.WithContext(ctx).First(&store, id).Error; err != nil {
		return StoreAccountDetail{}, err
	}
	return buildStoreAccountDetail(store), nil
}

func (s *StoreService) Upsert(ctx context.Context, req amazonReq.StoreAccountUpsertReq) (StoreAccountDetail, error) {
	if strings.TrimSpace(req.StoreName) == "" {
		return StoreAccountDetail{}, errors.New("storeName is required")
	}
	var store amazonModel.StoreAccount
	db := global.GVA_DB.WithContext(ctx)
	if req.ID > 0 {
		if err := db.First(&store, req.ID).Error; err != nil {
			return StoreAccountDetail{}, err
		}
	}
	store.StoreName = strings.TrimSpace(req.StoreName)
	store.Region = defaultString(strings.ToUpper(strings.TrimSpace(req.Region)), "NA")
	store.SellerID = strings.TrimSpace(req.SellerID)
	store.SellingPartnerID = strings.TrimSpace(req.SellingPartnerID)
	store.EnabledMarketplacesJSON = encodeJSON(uniqueStrings(req.EnabledMarketplaces))
	store.IsEnabled = req.IsEnabled
	if strings.TrimSpace(req.RefreshToken) != "" {
		encrypted, err := encryptAmazonSecret(strings.TrimSpace(req.RefreshToken))
		if err != nil {
			return StoreAccountDetail{}, err
		}
		store.RefreshTokenEncrypted = encrypted
		store.AuthStatus = "authorized"
		now := time.Now()
		store.LastAuthAt = &now
	}
	if err := db.Save(&store).Error; err != nil {
		return StoreAccountDetail{}, err
	}
	return buildStoreAccountDetail(store), nil
}

func (s *StoreService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("id is required")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&amazonModel.StoreAccount{}, id).Error
}

func (s *StoreService) AuthStart(ctx context.Context, id uint) (StoreAuthStartResult, error) {
	if id == 0 {
		return StoreAuthStartResult{}, errors.New("id is required")
	}
	var store amazonModel.StoreAccount
	if err := global.GVA_DB.WithContext(ctx).First(&store, id).Error; err != nil {
		return StoreAuthStartResult{}, err
	}
	state := randomStateToken()
	expiresAt := time.Now().Add(20 * time.Minute)
	if err := global.GVA_DB.WithContext(ctx).Model(&store).Updates(map[string]interface{}{
		"pending_auth_state":            state,
		"pending_auth_state_expired_at": &expiresAt,
	}).Error; err != nil {
		return StoreAuthStartResult{}, err
	}
	url, err := newSPAPIClient().buildAuthorizationURL(store, state)
	if err != nil {
		return StoreAuthStartResult{}, err
	}
	return StoreAuthStartResult{
		StoreID:      store.ID,
		AuthorizeURL: url,
		State:        state,
	}, nil
}

func (s *StoreService) AuthCallback(ctx context.Context, req amazonReq.StoreAccountAuthCallbackReq) (StoreAccountDetail, error) {
	if strings.TrimSpace(req.State) == "" || strings.TrimSpace(req.SpapiOAuthCode) == "" {
		return StoreAccountDetail{}, errors.New("state 和 spapi_oauth_code 必填")
	}
	var store amazonModel.StoreAccount
	if err := global.GVA_DB.WithContext(ctx).
		Where("pending_auth_state = ?", strings.TrimSpace(req.State)).
		First(&store).Error; err != nil {
		return StoreAccountDetail{}, err
	}
	if store.PendingAuthStateExpiredAt != nil && store.PendingAuthStateExpiredAt.Before(time.Now()) {
		return StoreAccountDetail{}, errors.New("授权状态已过期，请重新发起授权")
	}
	refreshToken, err := newSPAPIClient().exchangeAuthCode(ctx, req.SpapiOAuthCode)
	if err != nil {
		return StoreAccountDetail{}, err
	}
	encrypted, err := encryptAmazonSecret(refreshToken)
	if err != nil {
		return StoreAccountDetail{}, err
	}
	now := time.Now()
	updates := map[string]interface{}{
		"refresh_token_encrypted":       encrypted,
		"selling_partner_id":            defaultString(strings.TrimSpace(req.SellingPartnerID), store.SellingPartnerID),
		"auth_status":                   "authorized",
		"last_auth_at":                  &now,
		"pending_auth_state":            "",
		"pending_auth_state_expired_at": nil,
		"last_error":                    "",
	}
	if err := global.GVA_DB.WithContext(ctx).Model(&store).Updates(updates).Error; err != nil {
		return StoreAccountDetail{}, err
	}
	return s.Find(ctx, store.ID)
}

func (s *StoreService) TestConnection(ctx context.Context, id uint) (StoreConnectionTestResult, error) {
	if id == 0 {
		return StoreConnectionTestResult{}, errors.New("id is required")
	}
	var store amazonModel.StoreAccount
	if err := global.GVA_DB.WithContext(ctx).First(&store, id).Error; err != nil {
		return StoreConnectionTestResult{}, err
	}
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, "GET", "/sellers/v1/marketplaceParticipations", nil, nil, nil)
	if err != nil {
		_ = global.GVA_DB.WithContext(ctx).Model(&store).Update("last_error", err.Error()).Error
		return StoreConnectionTestResult{}, err
	}
	marketplaceCodes := extractMarketplaceCodes(resp)
	_ = global.GVA_DB.WithContext(ctx).Model(&store).Update("last_error", "").Error
	return StoreConnectionTestResult{
		StoreID:          id,
		Reachable:        true,
		MarketplaceCodes: marketplaceCodes,
	}, nil
}

func (s *StoreService) ListEnabledStores(ctx context.Context) ([]amazonModel.StoreAccount, error) {
	var stores []amazonModel.StoreAccount
	err := global.GVA_DB.WithContext(ctx).
		Where("is_enabled = ? AND auth_status = ?", true, "authorized").
		Order("id ASC").
		Find(&stores).Error
	return stores, err
}

func buildStoreAccountDetail(store amazonModel.StoreAccount) StoreAccountDetail {
	return StoreAccountDetail{
		ID:                        store.ID,
		StoreName:                 store.StoreName,
		Region:                    store.Region,
		SellerID:                  store.SellerID,
		SellingPartnerID:          store.SellingPartnerID,
		EnabledMarketplaces:       decodeStringJSON(store.EnabledMarketplacesJSON),
		AuthStatus:                store.AuthStatus,
		LastAuthAt:                formatCollectorTime(store.LastAuthAt),
		LastOrderSyncAt:           formatCollectorTime(store.LastOrderSyncAt),
		LastFBAInventorySyncAt:    formatCollectorTime(store.LastFBAInventorySyncAt),
		LastReturnSyncAt:          formatCollectorTime(store.LastReturnSyncAt),
		IsEnabled:                 store.IsEnabled,
		LastError:                 store.LastError,
		LastFBAInventorySyncError: store.LastFBAInventorySyncError,
		LastReturnSyncError:       store.LastReturnSyncError,
	}
}

func extractMarketplaceCodes(resp map[string]interface{}) []string {
	payload, _ := resp["payload"].(map[string]interface{})
	participations, _ := payload["payload"].([]interface{})
	if len(participations) == 0 {
		participations, _ = payload["marketplaceParticipations"].([]interface{})
	}
	result := make([]string, 0, len(participations))
	for _, entry := range participations {
		record, _ := entry.(map[string]interface{})
		marketplace, _ := record["marketplace"].(map[string]interface{})
		code := strings.TrimSpace(fmt.Sprintf("%v", marketplace["id"]))
		if code == "" {
			code = strings.TrimSpace(fmt.Sprintf("%v", marketplace["marketplaceId"]))
		}
		if code != "" {
			result = append(result, code)
		}
	}
	return uniqueStrings(result)
}

func findStoreByID(ctx context.Context, id uint) (amazonModel.StoreAccount, error) {
	var store amazonModel.StoreAccount
	if err := global.GVA_DB.WithContext(ctx).First(&store, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return store, errors.New("店铺不存在")
		}
		return store, err
	}
	return store, nil
}
