export type Product = {
  id: number
  name: string
  description: string
  price_fen: number
  status: number
  created_at: string
  updated_at: string
}

export type ProductListStatus = 1 | 2 | 'all'

export type ProductList = {
  products: Product[]
  total: number
  page: number
  page_size: number
}

export type Inventory = {
  id: number
  product_id: number
  stock_quantity: number
  created_at: string
  updated_at: string
}

export type InventoryRedisRebuildResult = {
  rebuild_count: number
}

export type InventoryRedisReconcileItem = {
  product_id: number
  mysql_quantity: number
  redis_quantity?: number | null
  status: string
}

export type InventoryRedisReconcileResult = {
  checked_count: number
  diff_count: number
  items: InventoryRedisReconcileItem[]
}

export type StockLog = {
  id: number
  product_id: number
  change_quantity: number
  before_quantity: number
  after_quantity: number
  biz_type: number
  // Carries the 63-bit Snowflake order id for biz_type 3/4 (order deduct /
  // rollback) and is serialized as a JSON string by the backend so that
  // JSON.parse does not truncate it past 2^53. Null for biz_type 1/2
  // (init / manual add), which have no business aggregate id.
  // Never convert back to number: that truncation is the defect itself.
  biz_id?: string | null
  remark: string
  created_at: string
}

export type OperationLog = {
  id: number
  user_id: number
  username: string
  action: string
  method: string
  path: string
  route: string
  http_status: number
  request_id: string
  client_ip: string
  user_agent: string
  created_at: string
}

export type OperationLogList = {
  operation_logs: OperationLog[]
  total: number
  page: number
  page_size: number
}

export type Order = {
  // 63-bit Snowflake ids are serialized as JSON strings by the backend so that
  // JSON.parse does not truncate them past 2^53 (GAP-012). Never convert back
  // to number: that truncation is the defect itself.
  id: string
  order_no: string
  total_amount_fen: number
  status: number
  paid_at?: string | null
  completed_at?: string | null
  cancelled_at?: string | null
  created_at: string
  updated_at: string
}

type OrderItem = {
  // `id` and `product_id` are auto-increment bigints and stay JSON numbers.
  // `order_id` carries the Snowflake order key and is serialized as a string
  // alongside Order.id (internal/model/order_items.go), so it is typed string.
  id: number
  order_id: string
  product_id: number
  product_name: string
  product_price_fen: number
  quantity: number
  subtotal_fen: number
  created_at: string
}

export type OrderDetail = {
  order: Order
  items: OrderItem[]
}

export type OrderList = {
  orders: Order[]
  total: number
  page: number
  page_size: number
}

export type CreateProductPayload = {
  name: string
  description: string
  price_fen: number
}

export type InitInventoryPayload = {
  product_id: number
  stock_quantity: number
}

export type AddInventoryPayload = {
  product_id: number
  quantity: number
}

export type CreateOrderPayload = {
  items: {
    product_id: number
    quantity: number
  }[]
}
