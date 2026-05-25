package amazon

const (
	supportCaseTypeBuyerMessage     = "buyer_message"
	supportCaseTypeAfterSales       = "after_sales"
	supportCaseTypeReturn           = "return"
	supportCaseTypeNegativeFeedback = "negative_feedback"
	supportCaseTypeAToZ             = "a_to_z"

	supportSourceTypeManual          = "manual"
	supportSourceTypeImport          = "import"
	supportSourceTypeAmazonMessaging = "amazon_messaging"

	supportReadStatusUnread = "unread"
	supportReadStatusRead   = "read"

	supportHandlingStatusPending    = "pending"
	supportHandlingStatusProcessing = "processing"
	supportHandlingStatusClosed     = "closed"

	supportDeliveryModeManualCopy   = "manual_copy"
	supportDeliveryModeAmazonDirect = "amazon_direct"

	supportMessageRoleCustomer = "customer"
	supportMessageRoleAgent    = "agent"
	supportMessageRoleInternal = "internal"

	supportMessageChannelManualCopy = "manual_copy"
	supportMessageChannelAmazon     = "amazon"
	supportMessageChannelImported   = "import"
	supportMessageChannelInternal   = "internal"

	supportSendStatusDraft          = "draft"
	supportSendStatusCopied         = "copied"
	supportSendStatusSent           = "sent"
	supportSendStatusFailed         = "failed"
	supportSendStatusFallbackManual = "fallback_manual"

	supportSLABucketNormal  = "normal"
	supportSLABucketWarning = "warning"
	supportSLABucketOverdue = "overdue"

	supportWarningLeadHours = 4
)
