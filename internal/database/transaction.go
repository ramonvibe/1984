package database

import (
 "context"
 "time"
 "github.com/jackc/pgx/v5/pgtype"
 "github.com/jackc/pgx/v5/pgxpool"
)

func Transaction(ctx context.Context, pool *pgxpool.Pool, work func(*Queries) error) error {
 tx, err := pool.Begin(ctx)
 if err != nil { return err }
 defer tx.Rollback(ctx)
 if err = work(New(tx)); err != nil { return err }
 return tx.Commit(ctx)
}

func ID(value int64) pgtype.Int8 { return pgtype.Int8{Int64:value, Valid:value>0} }
func Timestamp(value time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time:value, Valid:!value.IsZero()} }
