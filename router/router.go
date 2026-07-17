package router

import (
	"go-order-management-system/internal/auth"
	"go-order-management-system/internal/handler"
	"go-order-management-system/internal/middleware"
	"go-order-management-system/internal/observability"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Product      *handler.ProductHandler
	Inventory    *handler.InventoryHandler
	StockLog     *handler.StockLogHandler
	Order        *handler.OrderHandler
	Health       *handler.HealthHandler
	User         *handler.UserHandler
	OperationLog *handler.OperationLogHandler
	Audit        middleware.OperationAuditRecorder
	Metrics      *observability.Metrics
}

func SetupRouters(
	logger *slog.Logger,
	handlers Handlers,
	tokenManager *auth.TokenManager,
	roleChecker middleware.RoleChecker,
) *gin.Engine {
	r := gin.New()
	metrics := handlers.Metrics
	if metrics == nil {
		metrics = observability.NewMetrics()
	}

	r.Use(
		middleware.RequestID(),
		middleware.TraceContext(),
		middleware.HTTPMetrics(metrics.HTTP),
		middleware.AccessLog(logger),
		middleware.Recovery(logger),
	)

	registerHealthRouters(r, handlers, metrics)
	registerAPIRouter(r, handlers, tokenManager, roleChecker)
	return r
}

func registerHealthRouters(r *gin.Engine, handlers Handlers, metrics *observability.Metrics) {
	healthHandler := handlers.Health
	metricsHandler := handler.NewMetricsHandler(metrics)

	r.GET("/ping", healthHandler.PingHandler)
	r.GET("/live", healthHandler.LiveHandler)
	r.GET("/readyz", healthHandler.ReadyzHandler)
	r.GET("/metrics", metricsHandler.Prometheus)
}

func registerAPIRouter(
	rg *gin.Engine,
	handlers Handlers,
	tokenManager *auth.TokenManager,
	roleChecker middleware.RoleChecker,
) {
	apiV1 := rg.Group("/api/v1")

	registerAuthAPIRouter(apiV1, handlers.User)

	protected := apiV1.Group("")
	protected.Use(middleware.AuthMiddleware(tokenManager))
	registerUserAPIRouter(protected, handlers.User)
	registerProductAPIRouter(protected, handlers.Product, roleChecker, handlers.Audit)
	registerInventoryAPIRouter(protected, handlers.Inventory, roleChecker, handlers.Audit)
	registerStockLogAPIRouter(protected, handlers.StockLog, roleChecker, handlers.Audit)
	registerOperationLogAPIRouter(protected, handlers.OperationLog, roleChecker, handlers.Audit)
	registerOrderAPIRouter(protected, handlers.Order)
}

func registerAuthAPIRouter(rg *gin.RouterGroup, userHandler *handler.UserHandler) {
	authRoutes := rg.Group("/auth")
	authRoutes.POST("/register", userHandler.Register)
	authRoutes.POST("/login", userHandler.Login)
}

func registerUserAPIRouter(rg *gin.RouterGroup, userHandler *handler.UserHandler) {
	users := rg.Group("/users")
	users.GET("/me", userHandler.Me)
	users.PUT("/me/profile", userHandler.UpdateProfile)
	users.PATCH("/me/password", userHandler.UpdatePassword)
}

func registerProductAPIRouter(
	rg *gin.RouterGroup,
	productHandler *handler.ProductHandler,
	roleChecker middleware.RoleChecker,
	auditRecorder middleware.OperationAuditRecorder,
) {

	rg.GET("/products", productHandler.ListProducts)
	rg.GET("/products/:id", productHandler.GetProductByID)

	admin := rg.Group("")
	admin.Use(middleware.AdminMiddleware(roleChecker), middleware.OperationAudit(auditRecorder))
	admin.POST("/products", productHandler.CreateProduct)
	admin.PATCH("/products/:id/on-sale", productHandler.OnSaleProduct)
	admin.PATCH("/products/:id/off-sale", productHandler.OffSaleProduct)

}

func registerInventoryAPIRouter(
	rg *gin.RouterGroup,
	inventoryHandler *handler.InventoryHandler,
	roleChecker middleware.RoleChecker,
	auditRecorder middleware.OperationAuditRecorder,
) {

	rg.GET("/inventory/products/:product_id", inventoryHandler.GetInventoryByProductID)

	admin := rg.Group("")
	admin.Use(middleware.AdminMiddleware(roleChecker), middleware.OperationAudit(auditRecorder))
	admin.POST("/inventory/init", inventoryHandler.InitInventory)
	admin.POST("/inventory/add", inventoryHandler.AddInventory)
	admin.POST("/inventory/redis/rebuild", inventoryHandler.RebuildRedisInventoryStock)
	admin.GET("/inventory/redis/reconcile", inventoryHandler.ReconcileRedisInventoryStock)
}

func registerStockLogAPIRouter(
	rg *gin.RouterGroup,
	stockLogHandler *handler.StockLogHandler,
	roleChecker middleware.RoleChecker,
	auditRecorder middleware.OperationAuditRecorder,
) {

	admin := rg.Group("")
	admin.Use(middleware.AdminMiddleware(roleChecker), middleware.OperationAudit(auditRecorder))
	admin.GET("/stock-logs", stockLogHandler.ListStockLogs)

}

func registerOperationLogAPIRouter(
	rg *gin.RouterGroup,
	operationLogHandler *handler.OperationLogHandler,
	roleChecker middleware.RoleChecker,
	auditRecorder middleware.OperationAuditRecorder,
) {
	admin := rg.Group("")
	admin.Use(middleware.AdminMiddleware(roleChecker), middleware.OperationAudit(auditRecorder))
	admin.GET("/operation-logs", operationLogHandler.ListOperationLogs)
}

func registerOrderAPIRouter(rg *gin.RouterGroup, orderHandler *handler.OrderHandler) {

	rg.POST("/orders", orderHandler.CreateOrder)
	rg.GET("/orders/:id", orderHandler.GetOrderByID)
	rg.GET("/orders", orderHandler.ListOrders)
	rg.PATCH("/orders/:id/cancel", orderHandler.CancelOrders)
	rg.PATCH("/orders/:id/pay", orderHandler.PayOrder)
	rg.PATCH("/orders/:id/finish", orderHandler.FinishOrder)
}
