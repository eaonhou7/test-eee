package amazon

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestFinanceQuestionSaveFindAndUpdate(t *testing.T) {
	setupFinanceQuestionTestDB(t)
	service := new(FinanceQuestionService)

	created, err := service.Save(context.Background(), amazonReq.FinanceQuestionSaveReq{
		Title:        " 店铺创建资料 ",
		QuestionType: " 店铺创建 ",
		ContentHTML:  "<p>准备资料</p>",
	})
	if err != nil {
		t.Fatalf("save question: %v", err)
	}
	if created.ID == 0 || created.Title != "店铺创建资料" || created.QuestionType != "店铺创建" {
		t.Fatalf("unexpected created question: %+v", created)
	}

	found, err := service.Find(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("find question: %v", err)
	}
	if found.ContentHTML != "<p>准备资料</p>" {
		t.Fatalf("unexpected content: %+v", found)
	}

	updated, err := service.Save(context.Background(), amazonReq.FinanceQuestionSaveReq{
		ID:           created.ID,
		Title:        "收款账户验证",
		QuestionType: "收款账户",
		ContentHTML:  "<p>验证说明</p>",
	})
	if err != nil {
		t.Fatalf("update question: %v", err)
	}
	if updated.ID != created.ID || updated.Title != "收款账户验证" || updated.QuestionType != "收款账户" {
		t.Fatalf("unexpected updated question: %+v", updated)
	}
}

func TestFinanceQuestionListFiltersByTitleAndType(t *testing.T) {
	setupFinanceQuestionTestDB(t)
	service := new(FinanceQuestionService)
	seedFinanceQuestion(t, "店铺创建资料", "店铺创建")
	seedFinanceQuestion(t, "收款账户验证", "收款账户")
	seedFinanceQuestion(t, "收款账户回款", "收款账户")

	titleResult, err := service.List(context.Background(), amazonReq.FinanceQuestionListReq{
		PageInfo: commonReq.PageInfo{Page: 1, PageSize: 10},
		Title:    "账户",
	})
	if err != nil {
		t.Fatalf("list by title: %v", err)
	}
	if titleResult.Total != 2 || len(titleResult.List) != 2 {
		t.Fatalf("expected 2 title matches, got %+v", titleResult)
	}

	typeResult, err := service.List(context.Background(), amazonReq.FinanceQuestionListReq{
		PageInfo:     commonReq.PageInfo{Page: 1, PageSize: 10},
		QuestionType: "店铺创建",
	})
	if err != nil {
		t.Fatalf("list by type: %v", err)
	}
	if typeResult.Total != 1 || typeResult.List[0].Title != "店铺创建资料" {
		t.Fatalf("unexpected type result: %+v", typeResult)
	}
}

func TestFinanceQuestionSaveAddsCustomTypeOnce(t *testing.T) {
	setupFinanceQuestionTestDB(t)
	service := new(FinanceQuestionService)

	for i := 0; i < 2; i++ {
		if _, err := service.Save(context.Background(), amazonReq.FinanceQuestionSaveReq{
			Title:        "FBM 选品逻辑",
			QuestionType: " 选品 ",
			ContentHTML:  "<p>content</p>",
		}); err != nil {
			t.Fatalf("save custom type question: %v", err)
		}
	}

	var count int64
	if err := global.GVA_DB.Model(&amazonModel.FinanceQuestionType{}).Where("name = ?", "选品").Count(&count).Error; err != nil {
		t.Fatalf("count custom type: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one custom type, got %d", count)
	}
}

func TestFinanceQuestionListTypesIncludesDefaultsAndHistoricalTypes(t *testing.T) {
	setupFinanceQuestionTestDB(t)
	seedFinanceQuestion(t, "历史选品问题", "选品")

	types, err := new(FinanceQuestionService).ListTypes(context.Background())
	if err != nil {
		t.Fatalf("list types: %v", err)
	}

	for _, expected := range []string{"店铺创建", "收款账户", "选品"} {
		if !containsString(types, expected) {
			t.Fatalf("expected type %s in %+v", expected, types)
		}
	}
}

func TestFinanceQuestionFindMissingReturnsError(t *testing.T) {
	setupFinanceQuestionTestDB(t)

	_, err := new(FinanceQuestionService).Find(context.Background(), 999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record not found, got %v", err)
	}
}

func setupFinanceQuestionTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "finance-question.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&amazonModel.FinanceQuestion{}, &amazonModel.FinanceQuestionType{}); err != nil {
		t.Fatalf("migrate finance question table: %v", err)
	}
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func seedFinanceQuestion(t *testing.T, title string, questionType string) {
	t.Helper()
	if err := global.GVA_DB.Create(&amazonModel.FinanceQuestion{
		Title:        title,
		QuestionType: questionType,
		ContentHTML:  "<p>content</p>",
	}).Error; err != nil {
		t.Fatalf("seed finance question: %v", err)
	}
}
