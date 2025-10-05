package utils

import "errors"

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrFileTooLarge       = errors.New("file too large")
	ErrInvalidFileType    = errors.New("invalid file type")
	ErrFileUploadFailed   = errors.New("file upload failed")
	ErrFileDeleteFailed   = errors.New("file delete failed")
	ErrInvalidToken       = errors.New("invalid token")
	ErrProductNotFound    = errors.New("product not found")
)
