// internal/service/product_service.go
package service

import (
	"context"
	"strconv"

	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
	"github.com/DenisOzindzheDev/furniture-shop/internal/repository/postgres"
	"github.com/DenisOzindzheDev/furniture-shop/internal/repository/redis"
)

type ProductService struct {
	productRepo *postgres.ProductRepo
	cache       *redis.Cache
}

func NewProductService(productRepo *postgres.ProductRepo, cache *redis.Cache) *ProductService {
	return &ProductService{
		productRepo: productRepo,
		cache:       cache,
	}
}

func (s *ProductService) ListProducts(ctx context.Context, category string, page, pageSize int) ([]*entity.Product, error) {
	cacheKey := ""
	if category != "" {
		cacheKey = "products:" + category
	} else {
		cacheKey = "products:all"
	}

	offset := (page - 1) * pageSize

	// Try cache first
	var products []*entity.Product
	err := s.cache.Get(ctx, cacheKey, &products)
	if err == nil && products != nil {
		return products, nil
	}

	// Fallback to database
	products, err = s.productRepo.List(ctx, category, pageSize, offset)
	if err != nil {
		return nil, err
	}

	// Update cache
	go s.cache.Set(context.Background(), cacheKey, products)

	return products, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id int) (*entity.Product, error) {
	cacheKey := "product:" + strconv.Itoa(id)

	var product *entity.Product
	err := s.cache.Get(ctx, cacheKey, &product)
	if err == nil && product != nil {
		return product, nil
	}

	product, err = s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	go s.cache.Set(context.Background(), cacheKey, product)

	return product, nil
}
