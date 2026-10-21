package request

type CreateOrderRequest struct {
	IdempotencyKey string                   `json:"idempotency_key" binding:"required,max=128"`
	Items          []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type CreateOrderItemRequest struct {
	ProductID int64 `json:"product_id" binding:"required,gt=0"`
	Quantity  int64 `json:"quantity" binding:"required,gt=0"`
}

type CancelOrderRequest struct {
	OrderID int64 `json:"order_id" binding:"required,gt=0"`
}

type ListOrderRequest struct {
	Page     int `form:"page" binding:"gte=1,lte=1000000"`
	PageSize int `form:"page_size" binding:"gte=1,lte=100"`
}

type ListAdminOrdersRequest struct {
	UserID   *int64 `form:"user_id"`
	Status   *int   `form:"status"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=10"`
}

func (r *ListAdminOrdersRequest) NormalizePagination() {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.PageSize <= 0 {
		r.PageSize = 10
	} else if r.PageSize > 100 {
		r.PageSize = 100
	}
}
