package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (email, password, name, role) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		user.Email, user.Password, user.Name, user.Role).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `SELECT id, email, password, name, role, created_at, updated_at 
	          FROM users WHERE email = $1`

	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name,
		&user.Role, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int) (*entity.User, error) {
	query := `SELECT id, email, password, name, role, created_at, updated_at 
	          FROM users WHERE id = $1`

	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name,
		&user.Role, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}
