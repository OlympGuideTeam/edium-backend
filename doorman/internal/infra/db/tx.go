package db

import (
	"context"
	"database/sql"
)

type txKeyType struct{}

var txKey = txKeyType{}

func ContextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

func ExecutorFromContext(ctx context.Context, db *sql.DB) Executor {
	if tx, ok := ctx.Value(txKey).(*sql.Tx); ok {
		return tx
	}
	return db
}
