package http

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/DenisOzindzheDev/furniture-shop/internal/auth"
	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
	"github.com/DenisOzindzheDev/furniture-shop/internal/service"
	"github.com/DenisOzindzheDev/furniture-shop/pkg/utils"
)

// ProductAdminHandler handles product administration operations
// @Description ProductAdminHandler provides endpoints for product management by administrators
type ProductAdminHandler struct {
	productService *service.ProductService
}

func NewProductAdminHandler(productService *service.ProductService) *ProductAdminHandler {
	return &ProductAdminHandler{
		productService: productService,
	}
}

// CreateProductRequest represents the request body for creating a product
// @Description CreateProductRequest contains all required fields for product creation
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Stock       int     `json:"stock"`
}

// UpdateProductRequest represents the request body for updating a product
// @Description UpdateProductRequest contains optional fields for product update
type UpdateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Stock       int     `json:"stock"`
}

// ProductsResponse represents the response for product list operations
// @Description ProductsResponse contains paginated list of products with metadata
type ProductsResponse struct {
	Products []*entity.Product `json:"products"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	HasMore  bool              `json:"has_more"`
}

// CreateProduct godoc
// @Summary Создание нового продукта
// @Description Создает новый продукт с возможностью загрузки изображения. Требуются права администратора.
// @Tags admin-products
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string true "Название продукта"
// @Param description formData string true "Описание продукта"
// @Param price formData number true "Цена продукта"
// @Param category formData string true "Категория продукта"
// @Param stock formData integer true "Количество на складе"
// @Param image formData file false "Изображение продукта (JPEG, PNG, WebP до 10MB)"
// @Success 201 {object} entity.Product
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/products [post]
func (h *ProductAdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	category := r.FormValue("category")
	stockStr := r.FormValue("stock")

	if name == "" || description == "" || priceStr == "" || category == "" || stockStr == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		http.Error(w, "Invalid price", http.StatusBadRequest)
		return
	}

	stock, err := strconv.Atoi(stockStr)
	if err != nil {
		http.Error(w, "Invalid stock", http.StatusBadRequest)
		return
	}

	var imageFile io.ReadCloser
	var imageHeader *multipart.FileHeader

	file, header, err := r.FormFile("image")
	if err == nil {
		imageFile = file
		imageHeader = header
		defer imageFile.Close()
	}

	product := &entity.Product{
		Name:        name,
		Description: description,
		Price:       price,
		Category:    category,
		Stock:       stock,
	}

	if err := h.productService.CreateProduct(r.Context(), product, file, imageHeader); err != nil {
		switch err {
		case utils.ErrFileTooLarge:
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		case utils.ErrInvalidFileType:
			http.Error(w, "Invalid file type", http.StatusBadRequest)
		default:
			http.Error(w, "Failed to create product: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

// UpdateProduct godoc
// @Summary Обновление продукта
// @Description Обновляет существующий продукт. Все поля опциональны - обновляются только переданные поля. Требуются права администратора.
// @Tags admin-products
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID продукта"
// @Param name formData string false "Название продукта"
// @Param description formData string false "Описание продукта"
// @Param price formData number false "Цена продукта"
// @Param category formData string false "Категория продукта"
// @Param stock formData integer false "Количество на складе"
// @Param image formData file false "Изображение продукта (JPEG, PNG, WebP до 10MB)"
// @Success 200 {object} entity.Product
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/products/{id} [put]
func (h *ProductAdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	category := r.FormValue("category")
	stockStr := r.FormValue("stock")

	existingProduct, err := h.productService.GetProduct(r.Context(), id)
	if err != nil {
		if err == utils.ErrProductNotFound {
			http.Error(w, "Product not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get product", http.StatusInternalServerError)
		}
		return
	}

	if name != "" {
		existingProduct.Name = name
	}
	if description != "" {
		existingProduct.Description = description
	}
	if priceStr != "" {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}
		existingProduct.Price = price
	}
	if category != "" {
		existingProduct.Category = category
	}
	if stockStr != "" {
		stock, err := strconv.Atoi(stockStr)
		if err != nil {
			http.Error(w, "Invalid stock", http.StatusBadRequest)
			return
		}
		existingProduct.Stock = stock
	}

	var imageFile io.ReadCloser
	var imageHeader *multipart.FileHeader

	file, header, err := r.FormFile("image")
	if err == nil {
		imageFile = file
		imageHeader = header
		defer imageFile.Close()
	}

	if err := h.productService.UpdateProduct(r.Context(), existingProduct, file, imageHeader); err != nil {
		switch err {
		case utils.ErrFileTooLarge:
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		case utils.ErrInvalidFileType:
			http.Error(w, "Invalid file type", http.StatusBadRequest)
		case utils.ErrProductNotFound:
			http.Error(w, "Product not found", http.StatusNotFound)
		default:
			http.Error(w, "Failed to update product: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingProduct)
}

// DeleteProduct godoc
// @Summary Удаление продукта
// @Description Удаляет продукт по ID. Также удаляет связанное изображение из S3. Требуются права администратора.
// @Tags admin-products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID продукта"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/products/{id} [delete]
func (h *ProductAdminHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	if err := h.productService.DeleteProduct(r.Context(), id); err != nil {
		if err == utils.ErrProductNotFound {
			http.Error(w, "Product not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete product: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListProducts godoc
// @Summary Получение списка продуктов (админ)
// @Description Возвращает список продуктов с пагинацией для админ-панели. Требуются права администратора.
// @Tags admin-products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category query string false "Фильтр по категории"
// @Param page query int false "Номер страницы" minimum(1) default(1)
// @Param page_size query int false "Размер страницы" minimum(1) maximum(100) default(20)
// @Success 200 {object} ProductsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/products [get]
func (h *ProductAdminHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	category := r.URL.Query().Get("category")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	products, total, err := h.productService.ListProducts(r.Context(), category, page, pageSize)
	if err != nil {
		http.Error(w, "Failed to get products", http.StatusInternalServerError)
		return
	}

	hasMore := total > 0 && (page*pageSize) < total

	response := ProductsResponse{
		Products: products,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
