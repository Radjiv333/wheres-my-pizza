package domain

import "github.com/jackc/pgx/v5/pgxpool"

type Repository struct {
	Conn *pgxpool.Pool
}
