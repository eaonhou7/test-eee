package amazon

import (
	"context"
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
)

type FinanceQuestionService struct{}

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
	if strings.TrimSpace(req.QuestionType) == "" {
		return FinanceQuestionDetail{}, errors.New("question type is required")
	}
	var row amazonModel.FinanceQuestion
	db := global.GVA_DB.WithContext(ctx)
	if req.ID > 0 {
		if err := db.First(&row, req.ID).Error; err != nil {
			return FinanceQuestionDetail{}, err
		}
	}
	row.Title = strings.TrimSpace(req.Title)
	row.QuestionType = strings.TrimSpace(req.QuestionType)
	row.ContentHTML = strings.TrimSpace(req.ContentHTML)
	if err := db.Save(&row).Error; err != nil {
		return FinanceQuestionDetail{}, err
	}
	return buildFinanceQuestionDetail(row), nil
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
