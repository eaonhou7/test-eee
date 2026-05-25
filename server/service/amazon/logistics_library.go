package amazon

import (
	"context"

	amazonReq "github.com/flipped-aurora/gin-vue-admin/server/model/amazon/request"
)

var logisticsRepositoryApp = new(logisticsRepository)

func (s *LogisticsLibraryService) GetChannelPage(ctx context.Context, req amazonReq.LogisticsChannelPageReq) (LogisticsChannelPageResult, error) {
	return logisticsRepositoryApp.getChannelPage(ctx, req)
}

func (s *LogisticsLibraryService) GetChannelDetail(ctx context.Context, channelVersionID uint) (LogisticsChannelDetail, error) {
	return logisticsRepositoryApp.getChannelDetail(ctx, channelVersionID)
}

func (s *LogisticsLibraryService) GetRateRowPage(ctx context.Context, channelVersionID uint, page, pageSize int) (LogisticsRateRowPageResult, error) {
	return logisticsRepositoryApp.getRateRowPage(ctx, channelVersionID, page, pageSize)
}

func (s *LogisticsLibraryService) GetVersionPage(ctx context.Context, provider, logicalProductKey string, page, pageSize int) (LogisticsVersionPageResult, error) {
	return logisticsRepositoryApp.getVersionPage(ctx, provider, logicalProductKey, page, pageSize)
}
