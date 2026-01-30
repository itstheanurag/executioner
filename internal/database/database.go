package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/itstheanurag/executioner/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const DatabasePingTimeout = 10

type Database struct {
	Pool *pgxpool.Pool
	log  *zerolog.Logger
}

type multiTracer struct {
	tracers []any
}

func (mt *multiTracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryStart(
				ctx context.Context,
				conn *pgx.Conn,
				data pgx.TraceQueryStartData,
			) context.Context
		}); ok {
			ctx = t.TraceQueryStart(ctx, conn, data)
		}
	}

	return ctx
}

func (mt *multiTracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryEnd(
				ctx context.Context,
				conn *pgx.Conn,
				data pgx.TraceQueryEndData,
			)
		}); ok {
			t.TraceQueryEnd(ctx, conn, data)
		}
	}
}

func New(conf *config.Config, log *zerolog.Logger) (*Database, error) {
	host := net.JoinHostPort(conf.Db.Host, strconv.Itoa(conf.Db.Port))
	encodedPassword := url.QueryEscape(conf.Db.Password)

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		conf.Db.User,
		encodedPassword,
		host,
		conf.Db.Name,
		conf.Db.SSLMode,
	)

	pgxPoolConfig, err := pgxpool.ParseConfig(dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	pgxPoolConfig.ConnConfig.RuntimeParams["application_name"] = "executioner"

	pgxPoolConfig.ConnConfig.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		return dialer.DialContext(ctx, network, addr)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxPoolConfig)

	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTimeout*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("database connection established")

	return &Database{Pool: pool, log: log}, nil
}

func (db *Database) Close() error {
	db.log.Info().Msg("Closing database connection pool")
	db.Pool.Close()
	return nil
}
