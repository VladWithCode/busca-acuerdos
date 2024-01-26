package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect() (*pgxpool.Pool, error) {
    if Pool != nil {
        return Pool, nil
    }

    Pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
    if err != nil {
        return nil, err
    }

    return Pool, nil
}

