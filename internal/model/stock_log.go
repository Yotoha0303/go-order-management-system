package model

import "time"

const (
	StockBizInit          int8 = 1 //初始化库存
	StockBizManualAdd     int8 = 2 //手动入库
	StockBizOrderDeduct   int8 = 3 //下单扣减
	StockBizOrderRollback int8 = 4 //订单取消
)

type StockLog struct {
	ID             int64 `gorm:"primaryKey;autoIncrement;type:bigint" json:"id"`
	ProductID      int64 `gorm:"type:bigint;not null;index:idx_stock_logs_product_id" json:"product_id"`
	ChangeQuantity int64 `gorm:"type:bigint;not null" json:"change_quantity"`
	BeforeQuantity int64 `gorm:"type:bigint;not null" json:"before_quantity"`
	AfterQuantity  int64 `gorm:"type:bigint;not null" json:"after_quantity"`

	BizType int8 `gorm:"type:tinyint;not null;index:idx_stock_logs_biz_type" json:"biz_type"`
	// BizID carries the business aggregate id this log belongs to. It is nullable:
	// init / manual-add logs (StockBizInit, StockBizManualAdd) have no aggregate id.
	//
	// For StockBizOrderDeduct / StockBizOrderRollback it carries Order.ID, a 63-bit
	// Snowflake. It is serialized as a JSON string for the same reason as Order.ID:
	// an unquoted number is truncated by IEEE-754 double-precision parsers
	// (Number.MAX_SAFE_INTEGER = 2^53-1) on the consumer side. The `,string` option
	// applies to the pointed-to value, so a nil BizID still marshals as null.
	BizID *int64 `gorm:"type:bigint;index:idx_stock_logs_biz_id" json:"biz_id,string"`

	Remark    string    `gorm:"type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

func (StockLog) TableName() string {
	return "stock_logs"
}
