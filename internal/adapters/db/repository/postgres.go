package repository

import (
	"context"
	"fmt"

	"wheres-my-pizza/internal/core/domain"
	"wheres-my-pizza/pkg/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// "postgres://username:password@localhost:5432/database_name"
func NewRepository(cfg config.Config) (domain.Repository, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DatabaseName)
	conn, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return domain.Repository{}, err
	}
	// defer conn.Close(context.Background())
	return domain.Repository{Conn: conn}, nil
}
