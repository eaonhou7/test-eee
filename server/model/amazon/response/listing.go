package response

import amazonService "github.com/flipped-aurora/gin-vue-admin/server/service/amazon"

type ListingTemplatePageData = amazonService.ListingTemplatePageResult
type ListingTemplateDetailData = amazonService.ListingTemplateDetail
type ListingTemplateParseData = amazonService.ListingTemplateParseResult
type ListingFamilyPageData = amazonService.ListingFamilyPageResult
type ListingFamilyDetailData = amazonService.ListingFamilyDetail
type ListingTreePageData = amazonService.ListingTreePageResult
type ListingSaveData = amazonService.ListingSaveResult
type ListingValidationData = amazonService.ListingValidationResult
type ListingImageUploadData = amazonService.ListingImageUploadResult
type ListingExportData = amazonService.ListingExportTokenResult
