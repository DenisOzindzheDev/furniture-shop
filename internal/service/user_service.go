// internal/service/user_service.go
package service

import (
	"context"

	"github.com/DenisOzindzheDev/furniture-shop/internal/auth"
	"github.com/DenisOzindzheDev/furniture-shop/internal/common/errors"
	"github.com/DenisOzindzheDev/furniture-shop/internal/domain/entity"
	"github.com/DenisOzindzheDev/furniture-shop/internal/infra/kafka"
	"github.com/DenisOzindzheDev/furniture-shop/internal/infra/postgres"
)

type UserService struct {
	userRepo   *postgres.UserRepo
	jwtManager *auth.JWTManager
	producer   *kafka.Producer
}

func NewUserService(userRepo *postgres.UserRepo, jwtManager *auth.JWTManager, producer *kafka.Producer) *UserService {
	return &UserService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		producer:   producer,
	}
}

func (s *UserService) Register(ctx context.Context, user *entity.User) (string, error) {
	existing, err := s.userRepo.GetByEmail(ctx, user.Email)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return "", errors.ErrUserExists
	}

	if err := user.HashPassword(); err != nil {
		return "", err
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return "", err
	}

	go s.producer.SendEvent(context.Background(), kafka.EventUserRegistered, map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return s.jwtManager.Generate(user.ID, user.Email, user.Role)
}

func (s *UserService) Login(ctx context.Context, email, password string) (string, *entity.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}
	if user == nil || !user.CheckPassword(password) {
		return "", nil, errors.ErrInvalidCredentials
	}

	token, err := s.jwtManager.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}
func (s *UserService) GetProfile(ctx context.Context, userID int) (*entity.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}
