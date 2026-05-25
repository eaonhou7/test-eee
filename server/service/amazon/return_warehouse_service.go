package amazon

import (
	"context"
	"errors"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"gorm.io/gorm"
)

type ReturnWarehouseService struct{}

func (s *ReturnWarehouseService) List(ctx context.Context, req amazonReq.ReturnWarehouseListReq) (ReturnWarehousePageResult, error) {
	db := global.GVA_DB.WithContext(ctx).Model(&amazonModel.ReturnWarehouse{})
	if strings.TrimSpace(req.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		db = db.Where("name LIKE ? OR contact_name LIKE ? OR phone LIKE ?", keyword, keyword, keyword)
	}
	if strings.TrimSpace(req.CountryCode) != "" {
		db = db.Where("country_code = ?", strings.TrimSpace(strings.ToUpper(req.CountryCode)))
	}
	if req.IsEnabled != nil {
		db = db.Where("is_enabled = ?", *req.IsEnabled)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return ReturnWarehousePageResult{}, err
	}
	var rows []amazonModel.ReturnWarehouse
	if err := db.Scopes(req.PageInfo.Paginate()).Order("is_default DESC, priority ASC, id DESC").Find(&rows).Error; err != nil {
		return ReturnWarehousePageResult{}, err
	}
	result := ReturnWarehousePageResult{
		List:     make([]ReturnWarehouseDetail, 0, len(rows)),
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	for _, row := range rows {
		result.List = append(result.List, buildReturnWarehouseDetail(row))
	}
	return result, nil
}

func (s *ReturnWarehouseService) Find(ctx context.Context, id uint) (ReturnWarehouseDetail, error) {
	if id == 0 {
		return ReturnWarehouseDetail{}, errors.New("id is required")
	}
	var row amazonModel.ReturnWarehouse
	if err := global.GVA_DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return ReturnWarehouseDetail{}, err
	}
	return buildReturnWarehouseDetail(row), nil
}

func (s *ReturnWarehouseService) Upsert(ctx context.Context, req amazonReq.ReturnWarehouseUpsertReq) (ReturnWarehouseDetail, error) {
	if strings.TrimSpace(req.Name) == "" {
		return ReturnWarehouseDetail{}, errors.New("name is required")
	}
	if strings.TrimSpace(req.CountryCode) == "" {
		return ReturnWarehouseDetail{}, errors.New("countryCode is required")
	}
	var row amazonModel.ReturnWarehouse
	db := global.GVA_DB.WithContext(ctx)
	if req.ID > 0 {
		if err := db.First(&row, req.ID).Error; err != nil {
			return ReturnWarehouseDetail{}, err
		}
	}
	row.Name = strings.TrimSpace(req.Name)
	row.CountryCode = strings.TrimSpace(strings.ToUpper(req.CountryCode))
	row.SiteScopesJSON = encodeJSON(uniqueStrings(req.SiteScopes))
	row.ContactName = strings.TrimSpace(req.ContactName)
	row.Phone = strings.TrimSpace(req.Phone)
	row.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	row.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	row.AddressLine3 = strings.TrimSpace(req.AddressLine3)
	row.City = strings.TrimSpace(req.City)
	row.StateOrRegion = strings.TrimSpace(req.StateOrRegion)
	row.PostalCode = strings.TrimSpace(req.PostalCode)
	row.Priority = req.Priority
	row.IsDefault = req.IsDefault
	row.IsEnabled = req.IsEnabled
	if err := db.Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&amazonModel.ReturnWarehouse{}).Where("id <> ?", row.ID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&row).Error
	}); err != nil {
		return ReturnWarehouseDetail{}, err
	}
	return buildReturnWarehouseDetail(row), nil
}

func (s *ReturnWarehouseService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("id is required")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&amazonModel.ReturnWarehouse{}, id).Error
}

func buildReturnWarehouseDetail(row amazonModel.ReturnWarehouse) ReturnWarehouseDetail {
	return ReturnWarehouseDetail{
		ID:            row.ID,
		Name:          row.Name,
		CountryCode:   row.CountryCode,
		SiteScopes:    decodeStringJSON(row.SiteScopesJSON),
		ContactName:   row.ContactName,
		Phone:         row.Phone,
		AddressLine1:  row.AddressLine1,
		AddressLine2:  row.AddressLine2,
		AddressLine3:  row.AddressLine3,
		City:          row.City,
		StateOrRegion: row.StateOrRegion,
		PostalCode:    row.PostalCode,
		Priority:      row.Priority,
		IsDefault:     row.IsDefault,
		IsEnabled:     row.IsEnabled,
	}
}
