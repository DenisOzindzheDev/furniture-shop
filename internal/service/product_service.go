package service

import (
	"context"
	"mime/multipart"

	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
	"github.com/DenisOzindzheDev/furniture-shop/internal/repository/postgres"
	"github.com/DenisOzindzheDev/furniture-shop/internal/repository/redis"
	"github.com/DenisOzindzheDev/furniture-shop/pkg/utils"
)

type ProductService struct {
	productRepo  *postgres.ProductRepo
	imageSerivce *ImageService
	cache        *redis.Cache
}

func NewProductService(productRepo *postgres.ProductRepo, imageService *ImageService, cache *redis.Cache) *ProductService {
	return &ProductService{
		productRepo:  productRepo,
		imageSerivce: imageService,
		cache:        cache,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, product *entity.Product, imageFile multipart.File, imageHeader *multipart.FileHeader) error {
	if imageFile != nil && imageHeader != nil {
		imageURL, err := s.imageSerivce.UploadImage(ctx, imageFile, imageHeader)
		if err != nil {
			return err
		}
		product.ImageURL = imageURL
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		if product.ImageURL != "" {
			s.imageSerivce.DeleteImage(ctx, product.ImageURL)
		}
		return err
	}

	s.invalidateProductCache(ctx, product.Category, product.ID)

	return nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, product *entity.Product, imageFile multipart.File, imageHeader *multipart.FileHeader) error {
	oldProduct, err := s.productRepo.GetByID(ctx, product.ID)
	if err != nil {
		return err
	}
	if oldProduct == nil {
		return utils.ErrProductNotFound
	}

	if imageFile != nil && imageHeader != nil {
		imageURL, err := s.imageSerivce.UploadImage(ctx, imageFile, imageHeader)
		if err != nil {
			return err
		}

		if oldProduct.ImageURL != "" {
			s.imageSerivce.DeleteImage(ctx, oldProduct.ImageURL)
		}

		product.ImageURL = imageURL
	} else {
		product.ImageURL = oldProduct.ImageURL
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		if product.ImageURL != "" && product.ImageURL != oldProduct.ImageURL {
			s.imageSerivce.DeleteImage(ctx, product.ImageURL)
		}
		return err
	}

	s.invalidateProductCache(ctx, oldProduct.Category, product.ID)
	if oldProduct.Category != product.Category {
		s.invalidateProductCache(ctx, product.Category, product.ID)
	}

	return nil
}

// DeleteProduct удаляет продукт
func (s *ProductService) DeleteProduct(ctx context.Context, id int) error {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if product == nil {
		return utils.ErrProductNotFound
	}

	if product.ImageURL != "" {
		s.imageSerivce.DeleteImage(ctx, product.ImageURL)
	}

	if err := s.productRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.invalidateProductCache(ctx, product.Category, id)

	return nil
}

// invalidateProductCache инвалидирует кэш продуктов
func (s *ProductService) invalidateProductCache(ctx context.Context, category string, productID int) {
	s.cache.Delete(ctx, "products:all")
	s.cache.Delete(ctx, "products:"+category)
	s.cache.Delete(ctx, "product:"+string(rune(productID)))
}

// ListProducts возвращает список продуктов с пагинацией
func (s *ProductService) ListProducts(ctx context.Context, category string, page, pageSize int) ([]*entity.Product, int, error) {
	cacheKey := ""
	if category != "" {
		cacheKey = "products:" + category
	} else {
		cacheKey = "products:all"
	}

	offset := (page - 1) * pageSize

	var products []*entity.Product

	products, err := s.productRepo.List(ctx, category, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.productRepo.Count(ctx, category)
	if err != nil {
		return nil, 0, err
	}

	go s.cache.Set(context.Background(), cacheKey, products)

	return products, total, nil
}

// GetProduct возвращает продукт по ID
func (s *ProductService) GetProduct(ctx context.Context, id int) (*entity.Product, error) {
	cacheKey := "product:" + string(rune(id))

	var product *entity.Product
	err := s.cache.Get(ctx, cacheKey, &product)
	if err == nil && product != nil {
		return product, nil
	}

	product, err = s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, utils.ErrProductNotFound
	}

	go s.cache.Set(context.Background(), cacheKey, product)

	return product, nil
}

// UpdateStock обновляет количество товара на складе
func (s *ProductService) UpdateStock(ctx context.Context, id, stock int) error {
	if err := s.productRepo.UpdateStock(ctx, id, stock); err != nil {
		return err
	}

	s.cache.Delete(ctx, "product:"+string(rune(id)))

	return nil
}

// SearchProducts выполняет поиск продуктов
func (s *ProductService) SearchProducts(ctx context.Context, query string, page, pageSize int) ([]*entity.Product, error) {
	offset := (page - 1) * pageSize
	return s.productRepo.Search(ctx, query, pageSize, offset)
}
