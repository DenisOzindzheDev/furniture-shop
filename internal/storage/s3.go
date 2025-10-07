package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/DenisOzindzheDev/furniture-shop/internal/config"
	// каким-то хуем это deprecated либы стали
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Storage struct {
	client   *s3.S3
	uploader *s3manager.Uploader
	cfg      *config.AWS
	bucket   string
}

func NewS3Storage(cfg *config.AWS) (*S3Storage, error) {
	// Создаем кастомный HTTP клиент для MinIO
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	awsConfig := &aws.Config{
		Region:           aws.String(cfg.Region),
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		S3ForcePathStyle: aws.Bool(true),
		HTTPClient:       httpClient,
	}

	if cfg.S3Host != "" {
		endpoint := fmt.Sprintf("http://%s", cfg.S3Host)
		awsConfig.Endpoint = aws.String(endpoint)

		awsConfig.DisableSSL = aws.Bool(true)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	client := s3.New(sess)
	uploader := s3manager.NewUploader(sess)

	storage := &S3Storage{
		client:   client,
		uploader: uploader,
		cfg:      cfg,
		bucket:   cfg.S3Bucket,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := storage.ensureBucketExists(ctx); err != nil {
		return nil, fmt.Errorf("bucket check failed: %w", err)
	}

	return storage, nil
}

// ensureBucketExists проверяет и создает бакет если нужно
func (s *S3Storage) ensureBucketExists(ctx context.Context) error {
	_, err := s.client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})

	if err == nil {
		return nil
	}

	_, err = s.client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": "*"},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, s.bucket)

	_, err = s.client.PutBucketPolicyWithContext(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(s.bucket),
		Policy: aws.String(policy),
	})
	if err != nil {
		fmt.Printf("Warning: Failed to set bucket policy: %v\n", err)
	}

	return nil
}

// UploadFile загружает файл в S3
func (s *S3Storage) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	key := fmt.Sprintf("products/%s", filename)

	buffer := make([]byte, header.Size)
	_, err := file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	_, err = s.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String(header.Header.Get("Content-Type")),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	fileURL := s.generateFileURL(key)
	return fileURL, nil
}

// UploadBytes загружает байты в S3 (для тестов и других случаев)
func (s *S3Storage) UploadBytes(ctx context.Context, data []byte, filename, contentType string) (string, error) {
	key := fmt.Sprintf("products/%s", filename)

	_, err := s.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload bytes to S3: %w", err)
	}

	return s.generateFileURL(key), nil
}

// DeleteFile удаляет файл из S3
func (s *S3Storage) DeleteFile(ctx context.Context, fileURL string) error {
	key, err := s.extractKeyFromURL(fileURL)
	if err != nil {
		return err
	}

	_, err = s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// generateFileURL генерирует URL для файла
func (s *S3Storage) generateFileURL(key string) string {
	if s.cfg.S3Host != "" {
		// Для MinIO или кастомного S3
		return fmt.Sprintf("http://%s/%s/%s", s.cfg.S3Host, s.bucket, key)
	}

	// Для AWS S3 вообще я знаю что мы никогда там не захостим ничего, но пофиг пусть будет
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.cfg.Region, key)
}

// extractKeyFromURL извлекает ключ из URL
func (s *S3Storage) extractKeyFromURL(fileURL string) (string, error) {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse file URL: %w", err)
	}

	if s.cfg.S3Host != "" {
		path := strings.TrimPrefix(parsedURL.Path, "/")
		return strings.TrimPrefix(path, s.bucket+"/"), nil
	}

	return strings.TrimPrefix(parsedURL.Path, "/"), nil
}

// CheckBucket проверяет доступность бакета
func (s *S3Storage) CheckBucket(ctx context.Context) error {
	_, err := s.client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}
