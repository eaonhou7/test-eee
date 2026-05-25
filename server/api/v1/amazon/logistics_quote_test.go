package amazon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/songzhibin97/gkit/cache/local_cache"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestQuoteUSRouteHonorsJWTAndCasbin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := createCasbinTestDB(t)
	seedActiveQuoteData(t, db)

	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
	global.GVA_CONFIG = config.Server{
		JWT: config.JWT{
			SigningKey:  "test-signing-key",
			ExpiresTime: "7d",
			BufferTime:  "1d",
			Issuer:      "test-suite",
		},
		System: config.System{
			RouterPrefix: "",
		},
	}
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(time.Hour))

	router := gin.New()
	privateGroup := router.Group("")
	privateGroup.Use(middleware.JWTAuth(), middleware.CasbinHandler())
	api := LogisticsQuoteApi{}
	privateGroup.POST("/amazonLogistics/quoteUS", api.QuoteUS)

	unauthorized := submitQuoteRequest(t, router, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", unauthorized.Code)
	}

	allowed := submitQuoteRequest(t, router, createToken(t, 9528))
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected 200 for authority 9528, got %d", allowed.Code)
	}
	if !bytes.Contains(allowed.Body.Bytes(), []byte(`"code":0`)) {
		t.Fatalf("expected success payload, got %s", allowed.Body.String())
	}
	if !bytes.Contains(allowed.Body.Bytes(), []byte(`"overall_lowest"`)) {
		t.Fatalf("expected quote payload, got %s", allowed.Body.String())
	}
	if !bytes.Contains(allowed.Body.Bytes(), []byte(`"channel_version_id"`)) {
		t.Fatalf("expected channel version id in quote payload, got %s", allowed.Body.String())
	}

	denied := submitQuoteRequest(t, router, createToken(t, 8881))
	if denied.Code != http.StatusOK {
		t.Fatalf("expected 200 response wrapper for denied request, got %d", denied.Code)
	}
	if !bytes.Contains(denied.Body.Bytes(), []byte("权限不足")) {
		t.Fatalf("expected casbin deny message, got %s", denied.Body.String())
	}
}

func TestQuoteUSRouteUsesVolumetricWeightFormula(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := createCasbinTestDB(t)
	seedActiveQuoteData(t, db)

	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
	global.GVA_CONFIG = config.Server{
		JWT: config.JWT{
			SigningKey:  "test-signing-key",
			ExpiresTime: "7d",
			BufferTime:  "1d",
			Issuer:      "test-suite",
		},
		System: config.System{
			RouterPrefix: "",
		},
	}
	global.BlackCache = local_cache.NewCache(local_cache.SetDefaultExpire(time.Hour))

	router := gin.New()
	privateGroup := router.Group("")
	privateGroup.Use(middleware.JWTAuth(), middleware.CasbinHandler())
	api := LogisticsQuoteApi{}
	privateGroup.POST("/amazonLogistics/quoteUS", api.QuoteUS)

	body := bytes.NewBufferString(`{"weight_kg":0.2,"contains_battery":false,"length_cm":20,"width_cm":20,"height_cm":20}`)
	req := httptest.NewRequest(http.MethodPost, "/amazonLogistics/quoteUS", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-token", createToken(t, 9528))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"volumetric_weight_kg":1`)) {
		t.Fatalf("expected volumetric weight 1kg in response, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"billable_weight_kg":1`)) {
		t.Fatalf("expected billable weight 1kg in response, got %s", rec.Body.String())
	}
}

func createCasbinTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "casbin.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&gormadapter.CasbinRule{}, &amazonModel.LogisticsUploadBatch{}, &amazonModel.LogisticsChannelVersion{}, &amazonModel.LogisticsRateRowVersion{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	rules := []gormadapter.CasbinRule{
		{Ptype: "p", V0: "888", V1: "/amazonLogistics/quoteUS", V2: "POST"},
		{Ptype: "p", V0: "9528", V1: "/amazonLogistics/quoteUS", V2: "POST"},
	}
	if err := db.Create(&rules).Error; err != nil {
		t.Fatalf("seed casbin rules: %v", err)
	}
	return db
}

func seedActiveQuoteData(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now().UTC()

	yuntuBatch := amazonModel.LogisticsUploadBatch{Provider: "yuntu", SourceFileName: "yuntu-seed.xlsx", Status: "success"}
	if err := db.Create(&yuntuBatch).Error; err != nil {
		t.Fatalf("create yuntu batch: %v", err)
	}
	yuntuVersion := amazonModel.LogisticsChannelVersion{
		BatchID:             yuntuBatch.ID,
		Provider:            "yuntu",
		LogicalProductKey:   "yt100",
		ProductCode:         "YT100",
		ProductCodeType:     "产品代码",
		ChannelName:         "FBM SHIP+ 云途特快（普货）",
		SheetName:           "FBM SHIP+ 云途特快（普货）",
		LogisticsProvider:   "云途",
		ServiceCode:         "YT100",
		EffectiveAt:         &now,
		EffectiveTextRaw:    "2026-04-16 09:00",
		TransitTime:         "5-8工作日",
		CountryLabel:        "美国",
		RateKind:            "per_kg",
		VolumeDivisor:       8000,
		MinBillableWeightKG: 0.03,
		StepWeightKG:        0.01,
		IgnoreVolumetric:    true,
		IsActive:            true,
		ActivatedAt:         &now,
	}
	if err := db.Create(&yuntuVersion).Error; err != nil {
		t.Fatalf("create yuntu version: %v", err)
	}
	yuntuRow := amazonModel.LogisticsRateRowVersion{
		ChannelVersionID:   yuntuVersion.ID,
		SequenceNo:         1,
		WeightMinKG:        0,
		WeightMaxKG:        1,
		RatePerKG:          50,
		RegistrationFeeCNY: 8,
		MinBillableWeight:  0.03,
		TransitTime:        "5-8工作日",
	}
	if err := db.Create(&yuntuRow).Error; err != nil {
		t.Fatalf("create yuntu row: %v", err)
	}
}

func createToken(t *testing.T, authorityID uint) string {
	t.Helper()
	jwtTool := utils.NewJWT()
	token, err := jwtTool.CreateToken(jwtTool.CreateClaims(systemReq.BaseClaims{
		ID:          1,
		Username:    "tester",
		NickName:    "tester",
		AuthorityId: authorityID,
	}))
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return token
}

func submitQuoteRequest(t *testing.T, router *gin.Engine, token string) *httptest.ResponseRecorder {
	t.Helper()

	body := bytes.NewBufferString(`{"weight_kg":0.2,"contains_battery":false}`)
	req := httptest.NewRequest(http.MethodPost, "/amazonLogistics/quoteUS", body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("x-token", token)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
