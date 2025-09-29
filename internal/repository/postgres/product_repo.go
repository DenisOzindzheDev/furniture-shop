// internal/repository/postgres/product_repo.go
package postgres

import (
	"context"
	"database/sql"

	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) List(ctx context.Context, category string, limit, offset int) ([]*entity.Product, error) {
	query := `SELECT id, name, description, price, category, stock, image_url, created_at, updated_at 
	          FROM products WHERE ($1 = '' OR category = $1) LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, category, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var p entity.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category,
			&p.Stock, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		products = append(products, &p)
	}

	return products, nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id int) (*entity.Product, error) {
	query := `SELECT id, name, description, price, category, stock, image_url, created_at, updated_at 
	          FROM products WHERE id = $1`

	product := &entity.Product{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price,
		&product.Category, &product.Stock, &product.ImageURL,
		&product.CreatedAt, &product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return product, nil
}
