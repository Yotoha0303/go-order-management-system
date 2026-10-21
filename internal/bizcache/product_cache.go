package bizcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-order-management-system/internal/model"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type ProductCache struct {
	redisClient *redis.Client
}

func NewProductCache(redisClient *redis.Client) *ProductCache {
	return &ProductCache{
		redisClient: redisClient,
	}
}

const (
	ProductDetailCacheTTL     = 10 * time.Minute
	ProductDetailNullSentinel = "__NULL_PRODUCT__"
	ProductDetailNullCacheTTL = 60 * time.Second
)

func nullCacheTTLWithJitter() time.Duration {
	// 0 ~ 15 秒随机时间抖动，防止批量空值同时到期引发雪崩
	jitter := time.Duration(time.Now().UnixNano()%16) * time.Second
	if jitter < 0 {
		jitter = -jitter
	}
	return ProductDetailNullCacheTTL + jitter
}

func ProductDetailCacheKey(productID int64) string {
	return fmt.Sprintf("product:detail:%d", productID)
}

func (p *ProductCache) GetProductDetail(ctx context.Context, productID int64) (*model.Product, bool) {
	if p.redisClient == nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	val, err := p.redisClient.Get(ctx, ProductDetailCacheKey(productID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false
		}

		log.Printf("get product cache failed: product_id=%d err=%v", productID, err)
		return nil, false
	}

	// 命中了空值哨兵缓存：说明此商品不存在，返回 nil, true
	if val == ProductDetailNullSentinel {
		return nil, true
	}

	var product model.Product
	if err := json.Unmarshal([]byte(val), &product); err != nil {
		return nil, false
	}

	return &product, true
}

func (p *ProductCache) SetProductDetail(ctx context.Context, product *model.Product) {
	if p.redisClient == nil || product == nil {
		return
	}

	data, err := json.Marshal(product)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	_ = p.redisClient.Set(ctx, ProductDetailCacheKey(product.ID), data, ProductDetailCacheTTL).Err()
}

// SetProductDetailEmpty 向 Redis 写入空值哨兵，过期时间为短 TTL (60s + Jitter)，彻底杜绝非法 ID 穿透打爆 MySQL
func (p *ProductCache) SetProductDetailEmpty(ctx context.Context, productID int64) {
	if p.redisClient == nil || productID <= 0 {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	ttl := nullCacheTTLWithJitter()
	_ = p.redisClient.Set(ctx, ProductDetailCacheKey(productID), ProductDetailNullSentinel, ttl).Err()
}

func (p *ProductCache) DeleteProductDetailCache(ctx context.Context, productID int64) {
	if p.redisClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	_ = p.redisClient.Del(ctx, ProductDetailCacheKey(productID)).Err()
}
