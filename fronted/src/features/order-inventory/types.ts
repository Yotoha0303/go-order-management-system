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
  biz_id?: number | null
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
  id: number
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
  id: number
  order_id: number
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
