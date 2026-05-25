package amazon

import (
	"context"
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	exampleModel "github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/upload"
	"gorm.io/gorm"
)

type ImageService struct{}

func (s *ImageService) Upload(ctx context.Context, header *multipart.FileHeader) (ListingImageUploadResult, error) {
	if header == nil {
		return ListingImageUploadResult{}, errors.New("image file is required")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		return ListingImageUploadResult{}, errors.New("only jpg/jpeg/png/webp/gif image is supported")
	}
	if header.Size > 10*1024*1024 {
		return ListingImageUploadResult{}, errors.New("image size must be <= 10MB")
	}

	fileURL, fileKey, err := upload.NewOss().UploadFile(header)
	if err != nil {
		return ListingImageUploadResult{}, err
	}

	record := exampleModel.ExaFileUploadAndDownload{
		Name: header.Filename,
		Url:  fileURL,
		Tag:  strings.TrimPrefix(ext, "."),
		Key:  fileKey,
	}
	if err := global.GVA_DB.WithContext(ctx).Create(&record).Error; err != nil {
		return ListingImageUploadResult{}, err
	}

	return ListingImageUploadResult{
		FileID:   record.ID,
		FileName: record.Name,
		ImageURL: record.Url,
		FileKey:  record.Key,
	}, nil
}

func (s *ImageService) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("image id is required")
	}
	return global.GVA_DB.WithContext(ctx).Delete(&amazonModel.ListingItemImage{}, id).Error
}

func (s *ImageService) Sort(ctx context.Context, req amazonReq.SortListingImagesReq) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, image := range req.Images {
			updates := map[string]interface{}{
				"sort":       index + 1,
				"is_primary": image.IsPrimary,
				"slot_code":  strings.TrimSpace(image.SlotCode),
			}
			if err := tx.Model(&amazonModel.ListingItemImage{}).
				Where("id = ?", image.ID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
