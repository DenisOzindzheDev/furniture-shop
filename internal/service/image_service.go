package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/DenisOzindzheDev/furniture-shop/internal/config"
	"github.com/DenisOzindzheDev/furniture-shop/internal/storage"
	"github.com/DenisOzindzheDev/furniture-shop/pkg/utils"
)

type ImageService struct {
	storage *storage.S3Storage
	cfg     *config.Config
}

func NewImageService(storage *storage.S3Storage, cfg *config.Config) *ImageService {
	return &ImageService{
		storage: storage,
		cfg:     cfg,
	}
}

// ValidateImage проверяет изображение перед загрузкой
func (s *ImageService) ValidateImage(fileHeader *multipart.FileHeader) error {
	// Проверяем размер файла
	if fileHeader.Size > s.cfg.MaxUploadSize {
		return utils.ErrFileTooLarge
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if !s.isAllowedImageType(contentType) {
		return utils.ErrInvalidFileType
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !s.isAllowedExtension(ext) {
		return utils.ErrInvalidFileType
	}

	return nil
}

// UploadImage загружает изображение в S3
func (s *ImageService) UploadImage(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	if err := s.ValidateImage(header); err != nil {
		return "", err
	}

	fileURL, err := s.storage.UploadFile(ctx, file, header)
	if err != nil {
		return "", fmt.Errorf("%w: %v", utils.ErrFileUploadFailed, err)
	}

	return fileURL, nil
}

// DeleteImage удаляет изображение из S3
func (s *ImageService) DeleteImage(ctx context.Context, fileURL string) error {
	if fileURL == "" {
		return nil
	}

	if err := s.storage.DeleteFile(ctx, fileURL); err != nil {
		return fmt.Errorf("%w: %v", utils.ErrFileUploadFailed, err)
	}

	return nil
}

// isAllowedImageType проверяет разрешенный MIME type
func (s *ImageService) isAllowedImageType(contentType string) bool {
	for _, allowedType := range s.cfg.AllowedImageTypes {
		if contentType == allowedType {
			return true
		}
	}
	return false
}

// isAllowedExtension проверяет разрешенное расширение файла
func (s *ImageService) isAllowedExtension(ext string) bool {
	extMap := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
	}

	mimeType, exists := extMap[ext]
	if !exists {
		return false
	}

	return s.isAllowedImageType(mimeType)
}
