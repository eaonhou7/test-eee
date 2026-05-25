package request

import (
	"errors"
	"strings"

	commonReq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type LogisticsWorkbookUploadForm struct {
	Provider string `form:"provider" json:"provider"`
}

func (r LogisticsWorkbookUploadForm) Validate() error {
	switch strings.ToLower(strings.TrimSpace(r.Provider)) {
	case "yuntu", "yanwen", "santai":
		return nil
	default:
		return errors.New("provider must be yuntu, yanwen, or santai")
	}
}

type LogisticsChannelPageReq struct {
	commonReq.PageInfo
	Provider          string `json:"provider" form:"provider"`
	ActiveScope       string `json:"active_scope" form:"active_scope"`
	LogisticsProvider string `json:"logistics_provider" form:"logistics_provider"`
	Platform          string `json:"platform" form:"platform"`
	CountryLabel      string `json:"country_label" form:"country_label"`
	EffectiveDateFrom string `json:"effective_date_start" form:"effective_date_start"`
	EffectiveDateTo   string `json:"effective_date_end" form:"effective_date_end"`
	UploadedDateFrom  string `json:"uploaded_date_start" form:"uploaded_date_start"`
	UploadedDateTo    string `json:"uploaded_date_end" form:"uploaded_date_end"`
}

func (r LogisticsChannelPageReq) Validate() error {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.PageSize <= 0 {
		r.PageSize = 10
	}
	switch scope := strings.ToLower(strings.TrimSpace(r.ActiveScope)); scope {
	case "", "current", "history", "all":
		return nil
	default:
		return errors.New("active_scope must be current, history, or all")
	}
}

type LogisticsChannelDetailReq struct {
	ChannelVersionID uint `json:"channelVersionId" form:"channelVersionId"`
}

func (r LogisticsChannelDetailReq) Validate() error {
	if r.ChannelVersionID == 0 {
		return errors.New("channelVersionId is required")
	}
	return nil
}

type LogisticsRateRowPageReq struct {
	ChannelVersionID uint `json:"channelVersionId" form:"channelVersionId"`
	Page             int  `json:"page" form:"page"`
	PageSize         int  `json:"pageSize" form:"pageSize"`
}

func (r LogisticsRateRowPageReq) Validate() error {
	if r.ChannelVersionID == 0 {
		return errors.New("channelVersionId is required")
	}
	return nil
}

type LogisticsVersionPageReq struct {
	Provider          string `json:"provider" form:"provider"`
	LogicalProductKey string `json:"logical_product_key" form:"logical_product_key"`
	Page              int    `json:"page" form:"page"`
	PageSize          int    `json:"pageSize" form:"pageSize"`
}

func (r LogisticsVersionPageReq) Validate() error {
	if strings.TrimSpace(r.Provider) == "" {
		return errors.New("provider is required")
	}
	if strings.TrimSpace(r.LogicalProductKey) == "" {
		return errors.New("logical_product_key is required")
	}
	return nil
}
