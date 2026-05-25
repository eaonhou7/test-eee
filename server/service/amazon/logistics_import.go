package amazon

import (
	"context"
	"crypto/sha256"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/upload"
)

type logisticsServiceConfig struct {
	YuntuRateFile        string
	YanwenRateFile       string
	LogisticsUploadMaxMB int64
}

type LogisticsService struct {
	yuntuRateFile  string
	yanwenRateFile string
	uploadMaxBytes int64
}

type LogisticsLibraryService struct{}

var (
	logisticsServiceMu     sync.Mutex
	logisticsServiceCached *LogisticsService
	logisticsServiceState  logisticsServiceConfig
)

func NewLogisticsService(cfg logisticsServiceConfig) *LogisticsService {
	uploadMB := cfg.LogisticsUploadMaxMB
	if uploadMB <= 0 {
		uploadMB = 16
	}
	return &LogisticsService{
		yuntuRateFile:  strings.TrimSpace(cfg.YuntuRateFile),
		yanwenRateFile: strings.TrimSpace(cfg.YanwenRateFile),
		uploadMaxBytes: uploadMB * 1024 * 1024,
	}
}

func currentLogisticsService() *LogisticsService {
	logisticsServiceMu.Lock()
	defer logisticsServiceMu.Unlock()

	cfg := logisticsServiceConfig{
		YuntuRateFile:        global.GVA_CONFIG.Logistics.YuntuRateFile,
		YanwenRateFile:       global.GVA_CONFIG.Logistics.YanwenRateFile,
		LogisticsUploadMaxMB: global.GVA_CONFIG.Logistics.UploadMaxMB,
	}
	if logisticsServiceCached == nil || cfg != logisticsServiceState {
		logisticsServiceCached = NewLogisticsService(cfg)
		logisticsServiceState = cfg
	}
	return logisticsServiceCached
}

func (s *LogisticsService) UploadMaxBytes() int64 {
	return s.uploadMaxBytes
}

func (s *LogisticsService) parseProviderWorkbook(provider string, raw []byte, sourceMode, fileName string) (logisticsWorkbookData, error) {
	switch provider {
	case "yuntu":
		return parseYuntuWorkbook(raw, sourceMode, fileName)
	case "yanwen":
		return parseYanwenWorkbook(raw, sourceMode, fileName)
	case "santai":
		return parseSantaiWorkbook(raw, sourceMode, fileName)
	default:
		return logisticsWorkbookData{}, fmt.Errorf("unsupported provider %q", provider)
	}
}

func (s *LogisticsLibraryService) UploadMaxBytes() int64 {
	return currentLogisticsService().UploadMaxBytes()
}

func (s *LogisticsLibraryService) UploadWorkbook(ctx context.Context, provider string, header *multipart.FileHeader, uploadedBy, uploadedAuthorityID uint) (LogisticsWorkbookUploadResult, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != "yuntu" && provider != "yanwen" && provider != "santai" {
		return LogisticsWorkbookUploadResult{}, fmt.Errorf("provider must be yuntu, yanwen, or santai")
	}
	if header == nil {
		return LogisticsWorkbookUploadResult{}, fmt.Errorf("file is required")
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".xlsx" {
		return LogisticsWorkbookUploadResult{}, fmt.Errorf("file must be an .xlsx workbook")
	}

	raw, err := readMultipartFile(header)
	if err != nil {
		return LogisticsWorkbookUploadResult{}, err
	}
	if int64(len(raw)) > currentLogisticsService().UploadMaxBytes() {
		return LogisticsWorkbookUploadResult{}, fmt.Errorf("upload exceeds %d bytes", currentLogisticsService().UploadMaxBytes())
	}

	fileURL, fileKey, err := upload.NewOss().UploadFile(header)
	if err != nil {
		return LogisticsWorkbookUploadResult{}, err
	}

	sum := sha256.Sum256(raw)
	batch := amazonModel.LogisticsUploadBatch{
		Provider:            provider,
		SourceFileName:      header.Filename,
		FileURL:             fileURL,
		FileKey:             fileKey,
		FileSHA256:          fmt.Sprintf("%x", sum[:]),
		Status:              "processing",
		UploadedBy:          uploadedBy,
		UploadedAuthorityID: uploadedAuthorityID,
	}
	if err := logisticsRepositoryApp.createBatch(ctx, &batch); err != nil {
		return LogisticsWorkbookUploadResult{}, err
	}

	result := LogisticsWorkbookUploadResult{
		BatchID:        batch.ID,
		Provider:       provider,
		SourceFileName: header.Filename,
		Status:         "processing",
	}

	data, err := currentLogisticsService().parseProviderWorkbook(provider, raw, "upload", header.Filename)
	if err != nil {
		_ = logisticsRepositoryApp.markBatchFailed(ctx, batch.ID, err.Error())
		result.Status = "failed"
		result.FailureReason = err.Error()
		return result, err
	}

	parsedChannelCount, parsedRateRowCount, err := logisticsRepositoryApp.saveWorkbookImport(ctx, batch.ID, provider, data)
	if err != nil {
		_ = logisticsRepositoryApp.markBatchFailed(ctx, batch.ID, err.Error())
		result.Status = "failed"
		result.FailureReason = err.Error()
		return result, err
	}

	touchedProductCount := countTouchedProducts(data.Channels)
	if err := logisticsRepositoryApp.markBatchSuccess(ctx, batch.ID, parsedChannelCount, parsedRateRowCount, touchedProductCount); err != nil {
		return result, err
	}

	result.Status = "success"
	result.ParsedChannelCount = parsedChannelCount
	result.ParsedRateRowCount = parsedRateRowCount
	result.TouchedProductCount = touchedProductCount
	return result, nil
}

func readMultipartFile(header *multipart.FileHeader) ([]byte, error) {
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return parseUploadBytes(file)
}

func countTouchedProducts(channels []logisticsChannel) int {
	seen := map[string]struct{}{}
	for _, channel := range channels {
		key := normalizedText(channel.ServiceCode)
		if key == "" {
			key = normalizedText(channel.SheetName)
		}
		if key == "" {
			key = normalizedText(channel.ChannelName)
		}
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	return len(seen)
}
