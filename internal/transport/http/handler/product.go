package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/DenisOzindzheDev/furniture-shop/internal/auth"
	"github.com/DenisOzindzheDev/furniture-shop/internal/common/errors"
	"github.com/DenisOzindzheDev/furniture-shop/internal/domain/entity"
	"github.com/DenisOzindzheDev/furniture-shop/internal/service"
)

type ProductHandler struct {
	productService *service.ProductService
}

// ProductAdminHandler handles product administration operations
// @Description ProductAdminHandler provides endpoints for product management by administrators
type ProductAdminHandler struct {
	productService *service.ProductService
}

type ProductPDFHandler struct {
	productService *service.ProductService
	pdfService     *service.PDFService
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

// ErrorResponse represents a standard error response
// @Description Стандартный формат ответа при ошибке
type ErrorProductResponse struct {
	Code    int    `json:"code" example:"500"`
	Message string `json:"message" example:"Internal server error"`
	Details string `json:"details,omitempty" example:"ошибка подключения к базе"`
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

func NewProductAdminHandler(productService *service.ProductService) *ProductAdminHandler {
	return &ProductAdminHandler{
		productService: productService,
	}

}
func NewProductPDFHandler(productService *service.ProductService, pdfService *service.PDFService) *ProductPDFHandler {
	return &ProductPDFHandler{
		productService: productService,
		pdfService:     pdfService,
	}
}

// List products godoc
// @Summary Получение списка продуктов
// @Description Возвращает список продуктов с возможностью фильтрации по категории и пагинацией
// @Tags products
// @Accept json
// @Produce json
// @Param category query string false "Фильтр по категории"
// @Param page query int false "Номер страницы" default(1)
// @Param page_size query int false "Размер страницы" default(20)
// @Success 200 {object} ProductsResponse
// @Failure 500 {object} ErrorProductResponse
// @Router /products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
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
		writeProductError(w, http.StatusInternalServerError, "Не удалось получить список продуктов", err.Error())
		return
	}

	hasMore := total > 0 && (page*pageSize) < total

	writeJSON(w, http.StatusOK, ProductsResponse{
		Products: products,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	})
}

// GetProduct godoc
// @Summary Получение информации о продукте
// @Description Возвращает детальную информацию о продукте по ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "ID продукта"
// @Success 200 {object} entity.Product
// @Failure 400 {object} ErrorProductResponse
// @Failure 404 {object} ErrorProductResponse
// @Failure 500 {object} ErrorProductResponse
// @Router /products/{id} [get]
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeProductError(w, http.StatusBadRequest, "Некорректный ID продукта", err.Error())
		return
	}

	product, err := h.productService.GetProduct(r.Context(), id)
	if err != nil {
		if err == errors.ErrProductNotFound {
			writeProductError(w, http.StatusNotFound, "Продукт не найден", err.Error())
			return
		}
		writeProductError(w, http.StatusInternalServerError, "Ошибка при получении продукта", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, product)
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
// @Failure 400 {object} ErrorProductResponse
// @Failure 401 {object} ErrorProductResponse
// @Failure 403 {object} ErrorProductResponse
// @Failure 413 {object} ErrorProductResponse
// @Failure 500 {object} ErrorProductResponse
// @Router /admin/products [post]
func (h *ProductAdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		writeProductError(w, http.StatusForbidden, "Создать продукты может только администратор", errors.ErrInvalidToken.Error())
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeProductError(w, http.StatusBadRequest, "Ошибка запроса", err.Error())
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	category := r.FormValue("category")
	stockStr := r.FormValue("stock")

	if name == "" || description == "" || priceStr == "" || category == "" || stockStr == "" {
		writeProductError(w, http.StatusBadRequest, "Отсутствуют обязательные поля", "")
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		writeProductError(w, http.StatusBadRequest, "Некорректная цена", err.Error())
		return
	}

	stock, err := strconv.Atoi(stockStr)
	if err != nil {
		writeProductError(w, http.StatusBadRequest, "Некорректное количество", err.Error())
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		writeProductError(w, http.StatusBadRequest, "Ошибка чтения файла", err.Error())
		return
	}
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	product := &entity.Product{
		Name:        name,
		Description: description,
		Price:       price,
		Category:    category,
		Stock:       stock,
	}

	if err := h.productService.CreateProduct(r.Context(), product, file, header); err != nil {
		switch err {
		case errors.ErrFileTooLarge:
			writeProductError(w, http.StatusRequestEntityTooLarge, "Слишком большой файл", err.Error())
		case errors.ErrInvalidFileType:
			writeProductError(w, http.StatusBadRequest, "Недопустимый тип файла", err.Error())
		default:
			writeProductError(w, http.StatusInternalServerError, "Ошибка при создании продукта", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, product)
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
// @Failure 400 {object} ErrorProductResponse
// @Failure 401 {object} ErrorProductResponse
// @Failure 403 {object} ErrorProductResponse
// @Failure 404 {object} ErrorProductResponse
// @Failure 413 {object} ErrorProductResponse
// @Failure 500 {object} ErrorProductResponse
// @Router /admin/products/{id} [put]
func (h *ProductAdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		writeProductError(w, http.StatusForbidden, "Доступ запрещён", "только администратор может обновлять продукты")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeProductError(w, http.StatusBadRequest, "Некорректный ID продукта", err.Error())
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeProductError(w, http.StatusBadRequest, "Ошибка разбора формы", err.Error())
		return
	}

	existingProduct, err := h.productService.GetProduct(r.Context(), id)
	if err != nil {
		if err == errors.ErrProductNotFound {
			writeProductError(w, http.StatusNotFound, "Продукт не найден", err.Error())
			return
		}
		writeProductError(w, http.StatusInternalServerError, "Ошибка получения продукта", err.Error())
		return
	}

	if v := r.FormValue("name"); v != "" {
		existingProduct.Name = v
	}
	if v := r.FormValue("description"); v != "" {
		existingProduct.Description = v
	}
	if v := r.FormValue("category"); v != "" {
		existingProduct.Category = v
	}
	if v := r.FormValue("price"); v != "" {
		price, err := strconv.ParseFloat(v, 64)
		if err != nil {
			writeProductError(w, http.StatusBadRequest, "Некорректная цена", err.Error())
			return
		}
		existingProduct.Price = price
	}
	if v := r.FormValue("stock"); v != "" {
		stock, err := strconv.Atoi(v)
		if err != nil {
			writeProductError(w, http.StatusBadRequest, "Некорректное количество", err.Error())
			return
		}
		existingProduct.Stock = stock
	}

	file, header, err := r.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		writeProductError(w, http.StatusBadRequest, "Ошибка чтения файла", err.Error())
		return
	}
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	if err := h.productService.UpdateProduct(r.Context(), existingProduct, file, header); err != nil {
		switch err {
		case errors.ErrFileTooLarge:
			writeProductError(w, http.StatusRequestEntityTooLarge, "Слишком большой файл", err.Error())
		case errors.ErrInvalidFileType:
			writeProductError(w, http.StatusBadRequest, "Недопустимый тип файла", err.Error())
		case errors.ErrProductNotFound:
			writeProductError(w, http.StatusNotFound, "Продукт не найден", err.Error())
		default:
			writeProductError(w, http.StatusInternalServerError, "Ошибка при обновлении продукта", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, existingProduct)
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
// @Failure 400 {object} ErrorProductResponse
// @Failure 401 {object} ErrorProductResponse
// @Failure 403 {object} ErrorProductResponse
// @Failure 404 {object} ErrorProductResponse
// @Failure 500 {object} ErrorProductResponse
// @Router /admin/products/{id} [delete]
func (h *ProductAdminHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		writeProductError(w, http.StatusForbidden, "Доступ запрещён", "только администратор может удалять продукты")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeProductError(w, http.StatusBadRequest, "Некорректный ID продукта", err.Error())
		return
	}

	if err := h.productService.DeleteProduct(r.Context(), id); err != nil {
		if err == errors.ErrProductNotFound {
			writeProductError(w, http.StatusNotFound, "Продукт не найден", err.Error())
			return
		}
		writeProductError(w, http.StatusInternalServerError, "Ошибка при удалении продукта", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListProducts godoc (Admin)
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
// @Failure 401 {object} ErrorProductResponse
// @Failure 403 {object} ErrorProductResponse
// @Failure 500 {object} ErrorProductResponse
// @Router /admin/products [get]
func (h *ProductAdminHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		writeProductError(w, http.StatusForbidden, "Доступ запрещён", "только администратор может получать список продуктов")
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
		writeProductError(w, http.StatusInternalServerError, "Ошибка при получении списка продуктов", err.Error())
		return
	}

	hasMore := total > 0 && (page*pageSize) < total
	writeJSON(w, http.StatusOK, ProductsResponse{
		Products: products,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	})
}

// DownloadProductPDF godoc
// @Summary Скачать PDF карточку продукта
// @Description Генерирует и возвращает PDF файл с информацией о продукте
// @Tags products
// @Accept json
// @Produce application/pdf
// @Param id path int true "ID продукта"
// @Success 200 {file} file "PDF файл"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products/{id}/download [get]
func (h *ProductPDFHandler) DownloadProductPDF(w http.ResponseWriter, r *http.Request) {
	// Получаем ID продукта
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := h.productService.GetProduct(r.Context(), id)
	if err != nil {
		if err == errors.ErrProductNotFound {
			http.Error(w, "Product not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get product", http.StatusInternalServerError)
		}
		return
	}

	pdfBuffer, err := h.pdfService.GenerateProductPDF(product)
	if err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("product_%d_%s.pdf", product.ID, time.Now().Format("20060102"))

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", pdfBuffer.Len()))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBuffer.Bytes()))
}

// PreviewProductPDF godoc
// @Summary Просмотр PDF карточки продукта
// @Description Генерирует и возвращает PDF файл для просмотра в браузере
// @Tags products
// @Accept json
// @Produce application/pdf
// @Param id path int true "ID продукта"
// @Success 200 {file} file "PDF файл"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /products/{id}/preview [get]
func (h *ProductPDFHandler) PreviewProductPDF(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := h.productService.GetProduct(r.Context(), id)
	if err != nil {
		if err == errors.ErrProductNotFound {
			http.Error(w, "Product not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get product", http.StatusInternalServerError)
		}
		return
	}

	pdfBuffer, err := h.pdfService.GenerateProductPDF(product)
	if err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("product_%d_%s.pdf", product.ID, time.Now().Format("20060102"))

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", pdfBuffer.Len()))

	http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBuffer.Bytes()))
}

// TestPDF godoc
// @Summary Тест генерации PDF
// @Description Генерирует тестовый PDF для проверки функциональности
// @Tags development
// @Accept json
// @Produce application/pdf
// @Success 200 {file} file "Тестовый PDF файл"
// @Router /dev/test-pdf [get]
func (h *ProductPDFHandler) TestPDF(w http.ResponseWriter, r *http.Request) {
	testProduct := &entity.Product{
		ID:          1,
		Name:        "Тестовый диван",
		Description: "Это прекрасный угловой диван с механизмом трансформации. Идеально подходит для гостиной. Изготовлен из высококачественных материалов, обеспечивающих долговечность и комфорт.",
		Price:       29999.99,
		Category:    "Диваны",
		Stock:       15,
		ImageURL:    "https://via.placeholder.com/400x300/4A90E2/FFFFFF?text=Test+Product",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	pdfBuffer, err := h.pdfService.GenerateProductPDF(testProduct)
	if err != nil {
		http.Error(w, "Failed to generate test PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"test_product.pdf\"")

	http.ServeContent(w, r, "test_product.pdf", time.Now(), bytes.NewReader(pdfBuffer.Bytes()))
}
