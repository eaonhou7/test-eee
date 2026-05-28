package amazon

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type FinanceQuestionService struct{}

var defaultFinanceQuestionTypes = []amazonModel.FinanceQuestionType{
	{Name: "店铺创建", Sort: 1},
	{Name: "收款账户", Sort: 2},
}

func (s *FinanceQuestionService) List(ctx context.Context, req amazonReq.FinanceQuestionListReq) (FinanceQuestionPageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.FinanceQuestion{})
	if strings.TrimSpace(req.Title) != "" {
		keyword := "%" + strings.TrimSpace(req.Title) + "%"
		db = db.Where("title LIKE ?", keyword)
	}
	if strings.TrimSpace(req.QuestionType) != "" {
		db = db.Where("question_type = ?", strings.TrimSpace(req.QuestionType))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return FinanceQuestionPageResult{}, err
	}
	var rows []amazonModel.FinanceQuestion
	if err := db.Scopes(req.PageInfo.Paginate()).Order("id DESC").Find(&rows).Error; err != nil {
		return FinanceQuestionPageResult{}, err
	}
	result := FinanceQuestionPageResult{
		List:     make([]FinanceQuestionDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, buildFinanceQuestionDetail(row))
	}
	return result, nil
}

func (s *FinanceQuestionService) Find(ctx context.Context, id uint) (FinanceQuestionDetail, error) {
	if id == 0 {
		return FinanceQuestionDetail{}, errors.New("id is required")
	}
	var row amazonModel.FinanceQuestion
	if err := global.GVA_DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return FinanceQuestionDetail{}, err
	}
	return buildFinanceQuestionDetail(row), nil
}

func (s *FinanceQuestionService) Save(ctx context.Context, req amazonReq.FinanceQuestionSaveReq) (FinanceQuestionDetail, error) {
	if strings.TrimSpace(req.Title) == "" {
		return FinanceQuestionDetail{}, errors.New("title is required")
	}
	questionType := strings.TrimSpace(req.QuestionType)
	if questionType == "" {
		return FinanceQuestionDetail{}, errors.New("question type is required")
	}
	if utf8.RuneCountInString(questionType) > 64 {
		return FinanceQuestionDetail{}, errors.New("question type length must be <= 64")
	}
	var row amazonModel.FinanceQuestion
	db := global.GVA_DB.WithContext(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := ensureFinanceQuestionType(tx, questionType, 1000); err != nil {
			return err
		}
		if req.ID > 0 {
			if err := tx.First(&row, req.ID).Error; err != nil {
				return err
			}
		}
		row.Title = strings.TrimSpace(req.Title)
		row.QuestionType = questionType
		row.ContentHTML = strings.TrimSpace(req.ContentHTML)
		return tx.Save(&row).Error
	})
	if err != nil {
		return FinanceQuestionDetail{}, err
	}
	return buildFinanceQuestionDetail(row), nil
}

func (s *FinanceQuestionService) ListTypes(ctx context.Context) ([]string, error) {
	db := global.GVA_DB.WithContext(ctx)
	if err := EnsureFinanceQuestionTypes(db); err != nil {
		return nil, err
	}

	var rows []amazonModel.FinanceQuestionType
	if err := db.
		Where("name <> ''").
		Order("sort ASC, name ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.Name)
	}
	return result, nil
}

func EnsureFinanceQuestionTypes(db *gorm.DB) error {
	for _, item := range defaultFinanceQuestionTypes {
		if err := ensureFinanceQuestionType(db, item.Name, item.Sort); err != nil {
			return err
		}
	}

	var historicalTypes []string
	if err := db.Model(&amazonModel.FinanceQuestion{}).
		Where("question_type <> ''").
		Distinct().
		Pluck("question_type", &historicalTypes).Error; err != nil {
		return err
	}
	for _, questionType := range historicalTypes {
		if err := ensureFinanceQuestionType(db, questionType, 1000); err != nil {
			return err
		}
	}
	return nil
}

func ensureFinanceQuestionType(db *gorm.DB, name string, sort int) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if utf8.RuneCountInString(name) > 64 {
		return errors.New("question type length must be <= 64")
	}

	var row amazonModel.FinanceQuestionType
	err := db.Unscoped().Where("name = ?", name).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&amazonModel.FinanceQuestionType{Name: name, Sort: sort}).Error
	}

	updates := map[string]interface{}{"deleted_at": nil}
	if row.Sort == 0 && sort > 0 {
		updates["sort"] = sort
	}
	return db.Unscoped().Model(&row).Updates(updates).Error
}

func buildFinanceQuestionDetail(row amazonModel.FinanceQuestion) FinanceQuestionDetail {
	return FinanceQuestionDetail{
		ID:           row.ID,
		Title:        row.Title,
		QuestionType: row.QuestionType,
		ContentHTML:  row.ContentHTML,
		CreatedAt:    formatCollectorTime(&row.CreatedAt),
		UpdatedAt:    formatCollectorTime(&row.UpdatedAt),
	}
}
