package service

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"

	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
	"github.com/jung-kurt/gofpdf"
)

type PDFService struct {
	baseURL string
}

func NewPDFService(baseURL string) *PDFService {
	return &PDFService{
		baseURL: baseURL,
	}
}

func (s *PDFService) GenerateProductPDF(product *entity.Product) (*bytes.Buffer, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")

	s.addFirstPage(pdf, product)

	if product.Description != "" && len(product.Description) > 200 {
		pdf.AddPage()
		s.addDetailsPage(pdf, product)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return &buf, nil
}

func (s *PDFService) addFirstPage(pdf *gofpdf.Fpdf, product *entity.Product) {
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(0, 10, "Карточка продукта", "", 0, "C", false, 0, "")
		pdf.Ln(12)
	})

	if product.ImageURL != "" {
		s.addProductImage(pdf, product.ImageURL)
		pdf.Ln(10)
	}

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 8, product.Name, "", 1, "L", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 12)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 7, "Категория:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 7, product.Category, "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 7, "Цена:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 7, fmt.Sprintf("₽%.2f", product.Price), "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 7, "Наличие:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	stockText := fmt.Sprintf("%d шт.", product.Stock)
	if product.Stock == 0 {
		stockText = "Нет в наличии"
		pdf.SetTextColor(255, 0, 0)
	}
	pdf.CellFormat(0, 7, stockText, "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.Ln(5)

	if product.Description != "" {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 7, "Описание:", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 11)

		shortDesc := product.Description
		if len(shortDesc) > 200 {
			shortDesc = shortDesc[:200] + "..."
		}

		pdf.MultiCell(0, 5, shortDesc, "", "L", false)
	}

	pdf.Ln(10)
	s.addQRCode(pdf, product)

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 10, fmt.Sprintf("Сгенерировано %s", product.UpdatedAt.Format("02.01.2006")), "", 0, "C", false, 0, "")
	})
}

func (s *PDFService) addDetailsPage(pdf *gofpdf.Fpdf, product *entity.Product) {
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, "Детальное описание", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 6, product.Description, "", "L", false)

	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Дополнительная информация:", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 11)

	pdf.CellFormat(50, 6, "ID продукта:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("%d", product.ID), "", 1, "L", false, 0, "")

	pdf.CellFormat(50, 6, "Дата создания:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, product.CreatedAt.Format("02.01.2006 15:04"), "", 1, "L", false, 0, "")

	pdf.CellFormat(50, 6, "Обновлен:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 6, product.UpdatedAt.Format("02.01.2006 15:04"), "", 1, "L", false, 0, "")
}

func (s *PDFService) addProductImage(pdf *gofpdf.Fpdf, imageURL string) {
	imgData, err := s.downloadImage(imageURL)
	if err != nil {
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(0, 20, "[Изображение недоступно]", "", 1, "C", false, 0, "")
		return
	}

	imgReader := bytes.NewReader(imgData)
	opt := gofpdf.ImageOptions{
		ImageType: "JPG",
	}

	if strings.HasSuffix(strings.ToLower(imageURL), ".png") {
		opt.ImageType = "PNG"
	}

	imgName := fmt.Sprintf("product_img_%d", len(imageURL))

	pdf.RegisterImageOptionsReader(imgName, opt, imgReader)

	info := pdf.GetImageInfo(imgName)
	if info == nil {
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(0, 20, "[Ошибка загрузки изображения]", "", 1, "C", false, 0, "")
		return
	}

	width := info.Width()
	height := info.Height()
	maxWidth := 150.0

	if width > maxWidth {
		ratio := maxWidth / width
		width = maxWidth
		height = height * ratio
	}

	x := (210 - width) / 2 // A4 width = 210mm
	pdf.ImageOptions(imgName, x, pdf.GetY(), width, height, false, opt, 0, "")
	pdf.Ln(height + 5)
}

func (s *PDFService) addQRCode(pdf *gofpdf.Fpdf, product *entity.Product) {
	productURL := fmt.Sprintf("%s/products/%d", s.baseURL, product.ID)

	pdf.SetFont("Arial", "I", 9)
	pdf.CellFormat(0, 5, "Ссылка на продукт:", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(0, 4, productURL, "", 1, "C", false, 0, "")

	// Здесь можно добавить генерацию настоящего QR кода
	// Для этого понадобится дополнительная библиотека мб github.com/skip2/go-qrcode

	pdf.Ln(5)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 5, "[QR код будет здесь]", "", 1, "C", false, 0, "")
}

func (s *PDFService) downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *PDFService) ValidateImage(data []byte) error {
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid image: %w", err)
	}

	supportedFormats := map[string]bool{
		"jpeg": true,
		"jpg":  true,
		"png":  true,
	}

	if !supportedFormats[format] {
		return fmt.Errorf("unsupported image format: %s", format)
	}

	return nil
}

func (s *PDFService) OptimizeImage(data []byte, maxWidth int) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	// height := bounds.Dy()

	// Если изображение слишком большое, ресайзим
	if width > maxWidth {
		// newHeight := height * maxWidth / width
		// Здесь можно добавить ресайз изображения
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(&buf, img)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
