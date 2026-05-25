package amazon

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/xuri/excelize/v2"
)

type ExportService struct{}

type listingExportEntry struct {
	Family      ListingFamilyDetail
	Item        ListingItemDetail
	Marketplace ListingMarketplaceBinding
	Template    ListingTemplateDetail
}

func (s *ExportService) ExportSelected(ctx context.Context, req amazonReq.ListingExportSelectedDTO) (ListingExportTokenResult, error) {
	familyIDs, err := resolveSelectedFamilyIDs(ctx, req.FamilyIDs, req.ItemIDs)
	if err != nil {
		return ListingExportTokenResult{}, err
	}
	if len(familyIDs) == 0 {
		return ListingExportTokenResult{}, errors.New("请选择要导出的商品")
	}

	validation, err := (&ValidationService{}).validateFamilies(ctx, familyIDs, true)
	if err != nil {
		return ListingExportTokenResult{}, err
	}
	if !validation.Valid {
		return ListingExportTokenResult{}, errors.New(validation.Errors[0].Message)
	}

	templateCache := map[uint]ListingTemplateDetail{}
	grouped := map[string][]listingExportEntry{}
	order := make([]string, 0)

	for _, familyID := range familyIDs {
		family, err := (&ItemService{}).Find(ctx, familyID)
		if err != nil {
			return ListingExportTokenResult{}, err
		}
		for _, item := range family.Items {
			for _, marketplace := range item.Marketplaces {
				template, ok := templateCache[marketplace.TemplateID]
				if !ok {
					loaded, err := (&TemplateService{}).Find(ctx, marketplace.TemplateID)
					if err != nil {
						return ListingExportTokenResult{}, err
					}
					template = loaded
					templateCache[marketplace.TemplateID] = template
				}
				key := fmt.Sprintf("%d:%s", marketplace.TemplateID, marketplace.MarketplaceID)
				if _, ok := grouped[key]; !ok {
					order = append(order, key)
				}
				grouped[key] = append(grouped[key], listingExportEntry{
					Family:      family,
					Item:        item,
					Marketplace: marketplace,
					Template:    template,
				})
			}
		}
	}

	outputDir, publicBase, err := ensureAmazonExportDir()
	if err != nil {
		return ListingExportTokenResult{}, err
	}

	files := make([]string, 0, len(grouped))
	fileNames := make([]string, 0, len(grouped))
	for _, key := range order {
		entries := grouped[key]
		if len(entries) == 0 {
			continue
		}
		outputPath, fileName, err := generateAmazonWorkbook(outputDir, entries)
		if err != nil {
			return ListingExportTokenResult{}, err
		}
		files = append(files, outputPath)
		fileNames = append(fileNames, fileName)
	}
	if len(files) == 0 {
		return ListingExportTokenResult{}, errors.New("没有可导出的站点数据")
	}
	if len(files) == 1 {
		return ListingExportTokenResult{
			DownloadURL: pathToPublicURL(publicBase, fileNames[0]),
			FileName:    fileNames[0],
			IsZip:       false,
		}, nil
	}

	zipName := fmt.Sprintf("amazon-listings-%s.zip", time.Now().Format("20060102150405"))
	zipPath := filepath.Join(outputDir, zipName)
	if err := zipAmazonExports(zipPath, files, fileNames); err != nil {
		return ListingExportTokenResult{}, err
	}
	return ListingExportTokenResult{
		DownloadURL: pathToPublicURL(publicBase, zipName),
		FileName:    zipName,
		IsZip:       true,
	}, nil
}

func generateAmazonWorkbook(outputDir string, entries []listingExportEntry) (string, string, error) {
	template := entries[0].Template
	raw, err := (&TemplateService{}).resolveWorkbookBytes(context.Background(), template)
	if err != nil {
		return "", "", err
	}

	workbook, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	defer func() { _ = workbook.Close() }()

	sheetName := template.SheetName
	if strings.TrimSpace(sheetName) == "" {
		sheets := workbook.GetSheetList()
		if len(sheets) == 0 {
			return "", "", errors.New("模板工作簿为空")
		}
		sheetName = sheets[0]
	}

	fields := append([]ListingTemplateFieldRule{}, template.Fields...)
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].ColumnIndex < fields[j].ColumnIndex
	})

	rowIndex := maxInt(template.DataStartRowIndex, template.HeaderRowIndex+1)
	for _, entry := range sortExportEntries(entries) {
		for _, field := range fields {
			if !field.Enabled {
				continue
			}
			locale := selectExportLocale(field, entry.Marketplace)
			value := resolveFieldValue(entry.Family, entry.Item, entry.Marketplace, locale, field)
			if value == "" {
				value = strings.TrimSpace(field.DefaultValue)
			}
			if value == "" {
				level := strings.ToLower(strings.TrimSpace(field.RequiredLevel))
				if level != "" && level != "optional" && !strings.Contains(level, "选填") {
					return "", "", fmt.Errorf("导出失败，缺少必填字段 %s (%s)", field.FieldLabel, entry.Item.SKU)
				}
			}

			cell, err := excelize.CoordinatesToCellName(field.ColumnIndex, rowIndex)
			if err != nil {
				return "", "", err
			}
			if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
				return "", "", err
			}
		}
		rowIndex++
	}

	fileName := buildExportFileName(template, entries[0].Marketplace.SiteCode)
	outputPath := filepath.Join(outputDir, fileName)
	if err := workbook.SaveAs(outputPath); err != nil {
		return "", "", err
	}
	return outputPath, fileName, nil
}

func sortExportEntries(entries []listingExportEntry) []listingExportEntry {
	result := append([]listingExportEntry{}, entries...)
	sort.Slice(result, func(i, j int) bool {
		leftOrder := itemRoleOrder(result[i].Item.Role)
		rightOrder := itemRoleOrder(result[j].Item.Role)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		if result[i].Family.ID != result[j].Family.ID {
			return result[i].Family.ID < result[j].Family.ID
		}
		return result[i].Item.ID < result[j].Item.ID
	})
	return result
}

func selectExportLocale(field ListingTemplateFieldRule, marketplace ListingMarketplaceBinding) ListingLocaleData {
	if field.Scope != "locale" {
		return ListingLocaleData{}
	}
	if field.LocaleCode != "" {
		if locale, ok := findLocaleByCode(marketplace.Locales, field.LocaleCode); ok {
			return locale
		}
	}
	defaultLocale := map[string]string{
		"US": "en_US",
		"CA": "en_CA",
		"MX": "es_MX",
	}[strings.ToUpper(strings.TrimSpace(marketplace.SiteCode))]
	if locale, ok := findLocaleByCode(marketplace.Locales, defaultLocale); ok {
		return locale
	}
	if len(marketplace.Locales) > 0 {
		return marketplace.Locales[0]
	}
	return ListingLocaleData{}
}

func buildExportFileName(template ListingTemplateDetail, siteCode string) string {
	code := utils.HumpToUnderscore(strings.ReplaceAll(template.Code, "-", "_"))
	if code == "" {
		code = "amazon-template"
	}
	if siteCode == "" {
		siteCode = "NA"
	}
	return fmt.Sprintf("%s-%s-%s.xlsx", code, strings.ToLower(siteCode), time.Now().Format("20060102150405"))
}

func ensureAmazonExportDir() (string, string, error) {
	publicBase := filepath.Join(global.GVA_CONFIG.Local.StorePath, "amazon-exports")
	if err := os.MkdirAll(publicBase, os.ModePerm); err != nil {
		return "", "", err
	}
	return publicBase, publicBase, nil
}

func pathToPublicURL(publicBase, fileName string) string {
	path := filepath.Join(publicBase, fileName)
	return "/" + filepath.ToSlash(path)
}

func zipAmazonExports(zipPath string, files, fileNames []string) error {
	file, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	defer archive.Close()

	for index, sourcePath := range files {
		writer, err := archive.Create(fileNames[index])
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if _, err := writer.Write(raw); err != nil {
			return err
		}
	}
	return nil
}
