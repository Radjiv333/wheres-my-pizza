package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func GenerateOrderNumber(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
},
) (string, error) {
	day := time.Now().UTC().Format("2006-01-02")

	const sql = `
INSERT INTO order_number_seq(day, seq)
VALUES ($1::date, 1)
ON CONFLICT (day) DO UPDATE
  SET seq = order_number_seq.seq + 1
RETURNING seq;
`
	var seq int64
	if err := q.QueryRow(ctx, sql, day).Scan(&seq); err != nil {
		return "", fmt.Errorf("generate order seq: %w", err)
	}

	compactDay := strings.ReplaceAll(day, "-", "")
	return fmt.Sprintf("ORD_%s_%03d", compactDay, seq), nil
}
