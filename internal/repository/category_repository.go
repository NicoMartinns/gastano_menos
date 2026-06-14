package repository

import (
	"database/sql"
	"fmt"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) FindAll(userID string) ([]domain.Category, error) {
	query := `
		SELECT id, user_id, name, type, parent_id, created_at
		FROM categories
		WHERE user_id = $1
		ORDER BY parent_id NULLS FIRST, name ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Type, &c.ParentID, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

func (r *CategoryRepository) Create(c *domain.Category) error {
	query := `
		INSERT INTO categories (id, user_id, name, type, parent_id, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRow(query, c.UserID, c.Name, c.Type, c.ParentID).Scan(&c.ID, &c.CreatedAt)
}

func (r *CategoryRepository) Delete(id string, userID string) error {
	query := `DELETE FROM categories WHERE id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, id, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("categoria não encontrada")
	}

	return nil
}