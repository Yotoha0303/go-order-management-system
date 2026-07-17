package request

type ListOperationLogsRequest struct {
	UserID   *int64
	Action   string
	Page     int
	PageSize int
}
