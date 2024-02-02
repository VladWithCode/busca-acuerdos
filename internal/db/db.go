package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func Connect() (*pgxpool.Pool, error) {
	var err error
	DB, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	return DB, nil
}

func GetPool() (*pgxpool.Conn, error) {
	conn, err := DB.Acquire(context.Background())

	if err != nil {
		return nil, err
	}

	return conn, nil
}
