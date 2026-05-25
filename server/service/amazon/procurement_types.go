package amazon

type ProcurementTaskDetail struct {
	TaskToken    string                      `json:"taskToken"`
	OrderID      uint                        `json:"orderId"`
	GroupID      uint                        `json:"groupId"`
	ShopName     string                      `json:"shopName"`
	Status       string                      `json:"status"`
	TaskStatus   string                      `json:"taskStatus"`
	ErrorMessage string                      `json:"errorMessage"`
	Items        []ProcurementTaskItemDetail `json:"items"`
}

type ProcurementTaskItemDetail struct {
	GroupItemID          uint                   `json:"groupItemId"`
	OrderItemID          uint                   `json:"orderItemId"`
	SellerSKU            string                 `json:"sellerSku"`
	Title                string                 `json:"title"`
	ProductURL           string                 `json:"productUrl"`
	CollectedProductID   uint                   `json:"collectedProductId"`
	OfferID              string                 `json:"offerId"`
	ShopName             string                 `json:"shopName"`
	Selected1688SKUKey   string                 `json:"selected1688SkuKey"`
	Selected1688SKUAttrs map[string]interface{} `json:"selected1688SkuAttrs"`
	PurchaseQuantity     int                    `json:"purchaseQuantity"`
}
