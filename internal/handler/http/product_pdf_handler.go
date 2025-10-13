package http

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
	"github.com/DenisOzindzheDev/furniture-shop/internal/service"
	"github.com/DenisOzindzheDev/furniture-shop/pkg/utils"
)

type ProductPDFHandler struct {
	productService *service.ProductService
	pdfService     *service.PDFService
}

func NewProductPDFHandler(productService *service.ProductService, pdfService *service.PDFService) *ProductPDFHandler {
	return &ProductPDFHandler{
		productService: productService,
		pdfService:     pdfService,
	}
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
		if err == utils.ErrProductNotFound {
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
		if err == utils.ErrProductNotFound {
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
