package amazon

const (
	returnSummaryNone       = "none"
	returnSummaryOpen       = "open"
	returnSummaryProcessing = "processing"
	returnSummaryClosed     = "closed"
	returnSummaryException  = "exception"

	returnLinkPending     = "pending"
	returnLinkLinked      = "linked"
	returnLinkAmbiguous   = "ambiguous"
	returnLinkMissing     = "missing_order"
	returnLinkItemMissing = "missing_item"
	returnLinkManual      = "manual_review"

	returnDecisionGift         = "gift"
	returnDecisionWarehouse    = "warehouse"
	returnDecisionNewBuyer     = "new_buyer"
	returnDecisionManualReview = "manual_review"

	returnDecisionPending     = "pending"
	returnDecisionRecommended = "recommended"
	returnDecisionConfirmed   = "confirmed"
	returnDecisionClosed      = "closed"
	returnDecisionException   = "exception"

	returnDispositionPending   = "pending"
	returnDispositionCreated   = "created"
	returnDispositionCompleted = "completed"
	returnDispositionFailed    = "failed"
	returnDispositionReleased  = "released"

	returnTargetGift      = "gift"
	returnTargetWarehouse = "warehouse"
	returnTargetNewBuyer  = "new_buyer"

	supplySourceProcurement    = "procurement"
	supplySourceReturnRedirect = "return_redirect"

	returnRedirectStatusNone      = "none"
	returnRedirectStatusReserved  = "reserved"
	returnRedirectStatusBooked    = "booked"
	returnRedirectStatusReleased  = "released"
	returnRedirectStatusCompleted = "completed"
	returnRedirectStatusFailed    = "failed"

	orderWorkflowWaitingReturnRedirect = "fbm_waiting_return_redirect"
	orderWorkflowReturnRedirectShipped = "fbm_return_redirect_shipped"

	orderStatusSkipped                   = "skipped"
	logisticsStatusReturnRedirectPending = "return_redirect_pending"
	logisticsStatusReturnRedirectBooked  = "return_redirect_booked"

	returnGoodsValueBasisLandedCost = "landed_cost"
	returnQuoteModeManual           = "manual"
	returnQuoteModeAPI              = "api"
)
