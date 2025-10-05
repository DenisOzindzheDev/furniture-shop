package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DenisOzindzheDev/furniture-shop/internal/entity"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

// Create создает новый продукт
func (r *ProductRepo) Create(ctx context.Context, product *entity.Product) error {
	query := `
		INSERT INTO products (name, description, price, category, stock, image_url) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		product.Name,
		product.Description,
		product.Price,
		product.Category,
		product.Stock,
		product.ImageURL,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

// Update обновляет существующий продукт
func (r *ProductRepo) Update(ctx context.Context, product *entity.Product) error {
	query := `
		UPDATE products 
		SET name = $1, description = $2, price = $3, category = $4, stock = $5, image_url = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		product.Name,
		product.Description,
		product.Price,
		product.Category,
		product.Stock,
		product.ImageURL,
		product.ID,
	).Scan(&product.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("product not found: %d", product.ID)
		}
		return fmt.Errorf("update product: %w", err)
	}
	return nil
}

// Delete удаляет продукт по ID
func (r *ProductRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM products WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete product - get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product not found: %d", id)
	}

	return nil
}

// List возвращает список продуктов с пагинацией и фильтрацией
func (r *ProductRepo) List(ctx context.Context, category string, limit, offset int) ([]*entity.Product, error) {
	baseQuery := `
		SELECT id, name, description, price, category, stock, image_url, created_at, updated_at 
		FROM products`

	var query string
	var args []interface{}

	if category != "" {
		query = baseQuery + " WHERE category = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		args = []interface{}{category, limit, offset}
	} else {
		query = baseQuery + " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var p entity.Product
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.Category,
			&p.Stock,
			&p.ImageURL,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, &p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return products, nil
}

// GetByID возвращает продукт по ID
func (r *ProductRepo) GetByID(ctx context.Context, id int) (*entity.Product, error) {
	query := `
		SELECT id, name, description, price, category, stock, image_url, created_at, updated_at 
		FROM products WHERE id = $1`

	product := &entity.Product{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Category,
		&product.Stock,
		&product.ImageURL,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return product, nil
}

// Count возвращает общее количество продуктов (для пагинации)
func (r *ProductRepo) Count(ctx context.Context, category string) (int, error) {
	var query string
	var args []interface{}

	if category != "" {
		query = `SELECT COUNT(*) FROM products WHERE category = $1`
		args = []interface{}{category}
	} else {
		query = `SELECT COUNT(*) FROM products`
		args = []interface{}{}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count products: %w", err)
	}

	return count, nil
}

// UpdateStock обновляет количество товара на складе
func (r *ProductRepo) UpdateStock(ctx context.Context, id, stock int) error {
	query := `UPDATE products SET stock = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, stock, id)
	if err != nil {
		return fmt.Errorf("update product stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update product stock - get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product not found: %d", id)
	}

	return nil
}

// Search выполняет поиск продуктов по названию и описанию
func (r *ProductRepo) Search(ctx context.Context, query string, limit, offset int) ([]*entity.Product, error) {
	sqlQuery := `
		SELECT id, name, description, price, category, stock, image_url, created_at, updated_at 
		FROM products 
		WHERE name ILIKE $1 OR description ILIKE $1
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	searchPattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, sqlQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var p entity.Product
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.Category,
			&p.Stock,
			&p.ImageURL,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, &p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return products, nil
}
