package app

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"go-order-management-system/config"
	"go-order-management-system/internal/auth"
	"go-order-management-system/internal/bizcache"
	"go-order-management-system/internal/handler"
	"go-order-management-system/internal/inventoryreconcile"
	"go-order-management-system/internal/observability"
	"go-order-management-system/internal/ordertimeout"
	"go-order-management-system/internal/service"
	"go-order-management-system/pkg/database"
	"go-order-management-system/pkg/redis"
	"go-order-management-system/router"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Deps struct {
	Config             *config.Config
	DB                 *gorm.DB
	RedisClient        *goredis.Client
	Router             *gin.Engine
	Logger             *slog.Logger
	TokenManager       *auth.TokenManager
	OrderTimeoutWorker *ordertimeout.Worker
	InventoryWorker    *inventoryreconcile.Worker
	Metrics            *observability.Metrics
}

func InitDeps(logger *slog.Logger) (*Deps, error) {
	config.LoadEnv()

	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		return nil, err
	}

	db, err := database.InitDBWithLogger(cfg, logger)
	if err != nil {
		return nil, err
	}

	redisClient, err := redis.InitRedis(cfg)
	if err != nil {
		logger.Warn("redis unavailable, product cache disabled", "err", err)
	}

	metrics := observability.NewMetrics()
	productCache := bizcache.NewProductCache(redisClient)
	inventoryStockGuard := bizcache.NewInventoryStockGuardWithMetrics(redisClient, metrics.Business)

	productService := service.NewProductService(db, productCache)
	inventoryService := service.NewInventoryServiceWithStockWriter(db, inventoryStockGuard)
	var inventoryWorker *inventoryreconcile.Worker
	if cfg.InventoryReconcile.Enabled {
		inventoryWorker, err = inventoryreconcile.NewWorker(inventoryreconcile.Config{
			Interval: cfg.InventoryReconcile.Interval,
			Timeout:  cfg.InventoryReconcile.Timeout,
		}, inventoryService, logger)
		if err != nil {
			return nil, fmt.Errorf("build inventory reconcile worker: %w", err)
		}
	}
	stockLogService := service.NewStockLogService(db)
	operationLogService := service.NewOperationLogService(db)
	orderTimeoutConfig := cfg.RabbitMQ.OrderTimeout
	orderService := service.NewOrderServiceWithTimeoutInventoryPreDeductorAndMetrics(
		db,
		orderTimeoutConfig.Delay,
		inventoryStockGuard,
		metrics.Business,
	)
	orderTimeoutWorker, err := ordertimeout.NewWorker(ordertimeout.Config{
		URL:                cfg.RabbitMQ.URL,
		ConnectTimeout:     cfg.RabbitMQ.ConnectTimeout,
		ReconnectDelay:     cfg.RabbitMQ.ReconnectDelay,
		OutboxPollInterval: orderTimeoutConfig.OutboxPollInterval,
		OutboxRetryDelay:   orderTimeoutConfig.OutboxRetryDelay,
		PublishBatchSize:   orderTimeoutConfig.PublishBatchSize,
		ConsumerPrefetch:   orderTimeoutConfig.ConsumerPrefetch,
	}, db, orderService, logger)
	if err != nil {
		return nil, fmt.Errorf("build order timeout worker: %w", err)
	}
	userService := service.NewUserService(db)
	authorizationService := service.NewAuthorizationService(db)

	tokenManager, err := auth.NewTokenManager(
		os.Getenv("JWT_SECRET"),
		"go-order-management-system",
		time.Duration(cfg.JWT.ExpireHours)*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	handlers := router.Handlers{
		Product:      handler.NewProductHandler(productService),
		Inventory:    handler.NewInventoryHandler(inventoryService),
		StockLog:     handler.NewStockLogHandler(stockLogService),
		Order:        handler.NewOrderHandler(orderService),
		Health:       handler.NewHealthHandler(db),
		User:         handler.NewUserHandler(userService, tokenManager),
		OperationLog: handler.NewOperationLogHandler(operationLogService),
		Audit:        operationLogService,
		Metrics:      metrics,
	}

	r := router.SetupRouters(logger, handlers, tokenManager, authorizationService)

	return &Deps{
		Config:             cfg,
		DB:                 db,
		RedisClient:        redisClient,
		Router:             r,
		Logger:             logger,
		TokenManager:       tokenManager,
		OrderTimeoutWorker: orderTimeoutWorker,
		InventoryWorker:    inventoryWorker,
		Metrics:            metrics,
	}, nil
}
