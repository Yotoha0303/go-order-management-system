package service_test

import (
	"context"
	"database/sql"
	"errors"
	"go-order-management-system/internal/bizcache"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
	"go-order-management-system/internal/service"
	"os"
	"strings"
	"testing"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newProductService(t *testing.T) (*gorm.DB, *service.ProductService) {
	t.Helper()
	testDB := setupTestDB(t)
	cache := bizcache.NewProductCache(nil)
	productSvc := service.NewProductService(testDB, cache)
	return testDB, productSvc
}

type fakeProductCache struct {
	product          *model.Product
	hit              bool
	setProduct       *model.Product
	emptyProductID   int64
	deletedProductID int64
	emptyCallCount   int
	getCallCount     int
}

func (f *fakeProductCache) GetProductDetail(context.Context, int64) (*model.Product, bool) {
	f.getCallCount++
	return f.product, f.hit
}

func (f *fakeProductCache) SetProductDetail(_ context.Context, product *model.Product) {
	f.setProduct = product
}

func (f *fakeProductCache) SetProductDetailEmpty(_ context.Context, productID int64) {
	f.emptyProductID = productID
	f.emptyCallCount++
	f.product = nil
	f.hit = true
}

func (f *fakeProductCache) DeleteProductDetailCache(_ context.Context, productID int64) {
	f.deletedProductID = productID
}

func TestCreateProduct_Success(t *testing.T) {
	testDB, productSvc := newProductService(t)

	req := request.CreateProductRequest{
		Name:        "test product",
		Description: "desc",
		PriceFen:    199,
	}
	product, err := productSvc.CreateProduct(context.Background(), req)
	if err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	if product.ID <= 0 {
		t.Fatalf("expected product ID > 0,got %d", product.ID)
	}

	if product.Name != req.Name {
		t.Fatalf("expected name %q,got %q", req.Name, product.Name)
	}

	if product.Description != req.Description {
		t.Fatalf("expected description %q,got %q", req.Description, product.Description)
	}

	if product.PriceFen != req.PriceFen {
		t.Fatalf("expected price %d,got %d", req.PriceFen, product.PriceFen)
	}

	var saved model.Product

	if err := testDB.First(&saved, product.ID).Error; err != nil {
		t.Fatalf("query product record failed: %v", err)
	}

	if saved.Name != req.Name || saved.Description != req.Description || saved.PriceFen != req.PriceFen || saved.Status != model.ProductStatusOffSale {
		t.Fatalf("saved record mismatch, got %+v", saved)
	}

}

func TestCreateProduct_InvalidPrice(t *testing.T) {
	_, productSvc := newProductService(t)

	req := request.CreateProductRequest{
		Name:        "test product",
		Description: "desc",
		PriceFen:    0,
	}

	product, err := productSvc.CreateProduct(context.Background(), req)
	if !errors.Is(err, service.ErrInvalidProductPrice) {
		t.Fatalf("expected ErrInvalidProductPrice, got err=%v", err)
	}
	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}
}

func TestCreateProduct_SuccessTrimmed(t *testing.T) {
	_, productSvc := newProductService(t)

	req := request.CreateProductRequest{
		Name:        "  apple  ",
		Description: "  good  ",
		PriceFen:    199,
	}

	p, err := productSvc.CreateProduct(context.Background(), req)
	if err != nil {
		t.Fatalf("create product failed: %v", err)
	}
	if p.Name != "apple" || p.Description != "good" {
		t.Fatalf("trim failed, got name=%q description=%q", p.Name, p.Description)
	}
	if p.Status != model.ProductStatusOffSale {
		t.Fatalf("unexpected status: %d", p.Status)
	}
}

func TestCreateProduct_EmptyName(t *testing.T) {
	testDB, productSvc := newProductService(t)

	req := request.CreateProductRequest{
		Name:        "",
		Description: "name is empty",
		PriceFen:    199,
	}

	product, err := productSvc.CreateProduct(context.Background(), req)
	if !errors.Is(err, service.ErrInvalidProductName) {
		t.Fatalf("expected ErrInvalidProductName, got err=%v", err)
	}

	if product != nil {
		t.Fatalf("expected product nil, got %+v", product)
	}

	var count int64
	if err := testDB.Model(&model.Product{}).Where("description = ? AND name = ? ", req.Description, req.Name).Count(&count).Error; err != nil {
		t.Fatalf("count products failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 products, got %d", count)
	}
}

func TestCreateProduct_DescriptionTooLong(t *testing.T) {
	testDB, productSvc := newProductService(t)

	req := request.CreateProductRequest{
		Name:        "description-too-long-product",
		Description: strings.Repeat("a", 501),
		PriceFen:    199,
	}

	product, err := productSvc.CreateProduct(context.Background(), req)
	if !errors.Is(err, service.ErrInvalidProductDescription) {
		t.Fatalf("expect desciption less 500 character:,got %v", err)
	}

	if product != nil {
		t.Fatalf("expect product nil,got %+v", product)
	}

	var count int64
	if err := testDB.Model(&model.Product{}).Where("name = ? AND price_fen = ?", req.Name, 199).Count(&count).Error; err != nil {
		t.Fatalf("count products failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 products, got %d", count)
	}
}

func TestCreateProduct_DescriptionExactly500(t *testing.T) {
	testDB, productSvc := newProductService(t)

	req := request.CreateProductRequest{
		Name:        "description exactly 500",
		Description: strings.Repeat("a", 500),
		PriceFen:    199,
	}

	product, err := productSvc.CreateProduct(context.Background(), req)
	if err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	if product == nil {
		t.Fatal("expected product not nil")
		return
	}

	if product.ID <= 0 {
		t.Fatalf("expected product ID > 0, got %d", product.ID)
	}

	if product.Name != req.Name {
		t.Fatalf("expected name %q, got %q", req.Name, product.Name)
	}

	if product.PriceFen != req.PriceFen {
		t.Fatalf("expected price_fen %d, got %d", req.PriceFen, product.PriceFen)
	}

	if product.Description != req.Description {
		t.Fatalf("expected description %q, got %q", req.Description, product.Description)
	}

	if len(product.Description) != 500 {
		t.Fatalf("expected description length 500, got %d", len(product.Description))
	}

	if product.Status != model.ProductStatusOffSale {
		t.Fatalf("expected status off-sale, got %d", product.Status)
	}

	var saved model.Product
	if err := testDB.First(&saved, product.ID).Error; err != nil {
		t.Fatalf("query product failed: %v", err)
	}

	if saved.Name != req.Name || saved.Description != req.Description || saved.PriceFen != req.PriceFen || saved.Status != model.ProductStatusOffSale {
		t.Fatalf("saved record mismatch, got %+v", saved)
	}
}

func TestListProducts_OnlyOffSale(t *testing.T) {
	testDB, productSvc := newProductService(t)

	seedProduct(t, testDB, "off-sale", 100, model.ProductStatusOffSale)
	seedProduct(t, testDB, "on-sale", 100, model.ProductStatusOnSale)

	status := model.ProductStatusOffSale
	products, _, err := productSvc.ListProducts(context.Background(), request.ListProductsRequest{Status: &status})
	if err != nil {
		t.Fatalf("list products failed: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected 1 off-sale product, got %d", len(products))
	}
	if products[0].Status != model.ProductStatusOffSale {
		t.Fatalf("unexpected status: %d", products[0].Status)
	}
}

func TestListProducts_AllStatuses(t *testing.T) {
	testDB, productSvc := newProductService(t)

	seedProduct(t, testDB, "off-sale", 100, model.ProductStatusOffSale)
	seedProduct(t, testDB, "on-sale", 100, model.ProductStatusOnSale)

	products, _, err := productSvc.ListProducts(context.Background(), request.ListProductsRequest{})
	if err != nil {
		t.Fatalf("list all products failed: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
}

func TestListProducts_Pagination(t *testing.T) {
	testDB, productSvc := newProductService(t)
	seedProduct(t, testDB, "first", 100, model.ProductStatusOffSale)
	seedProduct(t, testDB, "second", 100, model.ProductStatusOffSale)
	seedProduct(t, testDB, "third", 100, model.ProductStatusOffSale)

	products, total, err := productSvc.ListProducts(context.Background(), request.ListProductsRequest{
		Page: 2, PageSize: 2,
	})
	if err != nil {
		t.Fatalf("list paginated products failed: %v", err)
	}
	if total != 3 || len(products) != 1 {
		t.Fatalf("expected total=3 and one product on page 2, got total=%d len=%d", total, len(products))
	}
}

func TestGetProductByID_NotFound(t *testing.T) {
	sqlDB := sql.OpenDB(&mockRRConnector{})
	defer sqlDB.Close()

	mockDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open mock gorm db: %v", err)
	}

	cache := &fakeProductCache{}
	productSvc := service.NewProductService(mockDB, cache)

	_, err = productSvc.GetProductByID(context.Background(), 99999)
	if !errors.Is(err, service.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestGetProductByID_CacheHit(t *testing.T) {
	expected := &model.Product{
		ID:       1001,
		Name:     "cached product",
		PriceFen: 100,
		Status:   model.ProductStatusOnSale,
	}
	cache := &fakeProductCache{product: expected, hit: true}
	productSvc := service.NewProductService(nil, cache)

	got, err := productSvc.GetProductByID(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("get cached product failed: %v", err)
	}
	if got != expected {
		t.Fatalf("expected cached product %+v, got %+v", expected, got)
	}
	if cache.setProduct != nil {
		t.Fatalf("cache hit should not set cache, got %+v", cache.setProduct)
	}
}

func TestGetProductByID_CacheMissSetsCache(t *testing.T) {
	testDB := setupTestDB(t)
	expected := seedProduct(t, testDB, "cache-miss-product", 100, model.ProductStatusOnSale)
	cache := &fakeProductCache{}
	productSvc := service.NewProductService(testDB, cache)

	got, err := productSvc.GetProductByID(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("get product failed: %v", err)
	}
	if got.ID != expected.ID {
		t.Fatalf("expected product ID %d, got %d", expected.ID, got.ID)
	}
	if cache.setProduct == nil || cache.setProduct.ID != expected.ID {
		t.Fatalf("expected product %d to be cached, got %+v", expected.ID, cache.setProduct)
	}
}

func TestGetProductByID_NullCacheHit(t *testing.T) {
	cache := &fakeProductCache{product: nil, hit: true}
	// 传入 nil db：若发生回源 DB 查询必触发 nil pointer panic，以此铁证零 DB 查询
	productSvc := service.NewProductService(nil, cache)

	got, err := productSvc.GetProductByID(context.Background(), 99999)
	if !errors.Is(err, service.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound on null cache hit, got err=%v", err)
	}
	if got != nil {
		t.Fatalf("expected nil product on null cache hit, got %+v", got)
	}
	if cache.getCallCount != 1 {
		t.Fatalf("expected 1 cache get call, got %d", cache.getCallCount)
	}
}

func TestGetProductByID_CachePenetrationPrevention_UnitMock(t *testing.T) {
	sqlDB := sql.OpenDB(&mockRRConnector{})
	defer sqlDB.Close()

	mockDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open mock gorm db: %v", err)
	}

	cache := &fakeProductCache{}
	productSvc := service.NewProductService(mockDB, cache)

	invalidID := int64(88888)

	// 第 1 次请求：缓存未命中，回源 DB（返回 ErrRecordNotFound），触发 SetProductDetailEmpty
	got1, err1 := productSvc.GetProductByID(context.Background(), invalidID)
	if !errors.Is(err1, service.ErrProductNotFound) {
		t.Fatalf("1st call: expected ErrProductNotFound, got %v", err1)
	}
	if got1 != nil {
		t.Fatalf("1st call: expected nil product, got %+v", got1)
	}
	if cache.emptyCallCount != 1 {
		t.Fatalf("1st call: expected SetProductDetailEmpty called once, got %d", cache.emptyCallCount)
	}
	if cache.emptyProductID != invalidID {
		t.Fatalf("1st call: expected emptyProductID=%d, got %d", invalidID, cache.emptyProductID)
	}

	// 此时 fakeProductCache 已模拟写入空值哨兵 (hit=true, product=nil)
	// 第 2 次请求：直接命中 Redis 空值哨兵，零 DB 查询阻断穿透
	// 为绝对证明零 DB 访问，将 productSvc 的 db 替换为 nil，若发生 DB 查询必触发 panic
	nilDBSvc := service.NewProductService(nil, cache)
	got2, err2 := nilDBSvc.GetProductByID(context.Background(), invalidID)
	if !errors.Is(err2, service.ErrProductNotFound) {
		t.Fatalf("2nd call: expected ErrProductNotFound, got %v", err2)
	}
	if got2 != nil {
		t.Fatalf("2nd call: expected nil product, got %+v", got2)
	}
	if cache.getCallCount != 2 {
		t.Fatalf("expected total 2 cache get calls, got %d", cache.getCallCount)
	}
}

func TestGetProductByID_CachePenetrationPrevention_IntegrationMySQL(t *testing.T) {
	if os.Getenv("RUN_MYSQL_TEST") != "1" {
		t.Skip("skip MySQL integration test; set RUN_MYSQL_TEST=1 to run")
	}

	testDB := setupTestDB(t)
	cache := &fakeProductCache{}
	productSvc := service.NewProductService(testDB, cache)

	invalidID := int64(88888)

	// 第 1 次请求：缓存未命中，回源真实 MySQL 返回 ErrRecordNotFound，触发 SetProductDetailEmpty
	got1, err1 := productSvc.GetProductByID(context.Background(), invalidID)
	if !errors.Is(err1, service.ErrProductNotFound) {
		t.Fatalf("1st call: expected ErrProductNotFound, got %v", err1)
	}
	if got1 != nil {
		t.Fatalf("1st call: expected nil product, got %+v", got1)
	}
	if cache.emptyCallCount != 1 {
		t.Fatalf("1st call: expected SetProductDetailEmpty called once, got %d", cache.emptyCallCount)
	}
	if cache.emptyProductID != invalidID {
		t.Fatalf("1st call: expected emptyProductID=%d, got %d", invalidID, cache.emptyProductID)
	}

	// 第 2 次请求：零 DB 查询，直接命中空值哨兵
	nilDBSvc := service.NewProductService(nil, cache)
	got2, err2 := nilDBSvc.GetProductByID(context.Background(), invalidID)
	if !errors.Is(err2, service.ErrProductNotFound) {
		t.Fatalf("2nd call: expected ErrProductNotFound, got %v", err2)
	}
	if got2 != nil {
		t.Fatalf("2nd call: expected nil product, got %+v", got2)
	}
	if cache.getCallCount != 2 {
		t.Fatalf("expected total 2 cache get calls, got %d", cache.getCallCount)
	}
}

func TestOnSaleProduct_Success(t *testing.T) {
	testDB := setupTestDB(t)
	cache := &fakeProductCache{}
	productSvc := service.NewProductService(testDB, cache)

	p := seedProduct(t, testDB, "p1", 100, model.ProductStatusOffSale)
	if err := productSvc.OnSaleProduct(context.Background(), p.ID); err != nil {
		t.Fatalf("on sale failed: %v", err)
	}

	var got model.Product
	if err := testDB.First(&got, p.ID).Error; err != nil {
		t.Fatalf("query product failed: %v", err)
	}
	if got.Status != model.ProductStatusOnSale {
		t.Fatalf("expected on-sale status, got %d", got.Status)
	}
	if cache.deletedProductID != p.ID {
		t.Fatalf("expected cache for product %d to be deleted, got %d", p.ID, cache.deletedProductID)
	}
}

func TestOffSaleProduct_Success(t *testing.T) {
	testDB := setupTestDB(t)
	cache := &fakeProductCache{}
	productSvc := service.NewProductService(testDB, cache)

	p := seedProduct(t, testDB, "p1", 100, model.ProductStatusOnSale)
	if err := productSvc.OffSaleProduct(context.Background(), p.ID); err != nil {
		t.Fatalf("off sale failed: %v", err)
	}

	var got model.Product
	if err := testDB.First(&got, p.ID).Error; err != nil {
		t.Fatalf("query product failed: %v", err)
	}
	if got.Status != model.ProductStatusOffSale {
		t.Fatalf("expected off-sale status, got %d", got.Status)
	}
	if cache.deletedProductID != p.ID {
		t.Fatalf("expected cache for product %d to be deleted, got %d", p.ID, cache.deletedProductID)
	}
}
