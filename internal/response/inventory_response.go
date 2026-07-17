package response

type InventoryRedisRebuildResponse struct {
	RebuildCount int `json:"rebuild_count"`
}

type InventoryRedisReconcileResponse struct {
	CheckedCount int                           `json:"checked_count"`
	DiffCount    int                           `json:"diff_count"`
	Items        []InventoryRedisReconcileItem `json:"items"`
}

type InventoryRedisReconcileItem struct {
	ProductID     int64  `json:"product_id"`
	MySQLQuantity int64  `json:"mysql_quantity"`
	RedisQuantity *int64 `json:"redis_quantity"`
	Status        string `json:"status"`
}
