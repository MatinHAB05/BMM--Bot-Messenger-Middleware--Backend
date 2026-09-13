// Package database owns the PostgreSQL connection lifecycle via GORM.
// Schema changes are applied separately by Goose (see bootstrap/init.go);
// this package is only responsible for the connection pool itself.
package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Host          string
	Port          string
	User          string
	Password      string
	DBName        string
	SSLMode       string
	MaxOpenDbConn int
	MaxIdleDbConn time.Duration
	MaxDbLifeTime time.Duration
}

type Database interface {
	GetGormDB() *gorm.DB
	WithTx(tx *gorm.DB) Database
	GetDBConfig() Config
}

type PostgresDatabase struct {
	DB *gorm.DB
	*Config
}

var (
	dbOnce     sync.Once
	dbInstance *PostgresDatabase
	dbErr      error
)

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func NewPostgresDatabase(ctx context.Context, dbConfig *Config) (Database, error) {
	dbOnce.Do(func() {
		dsn := dbConfig.DSN()

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			dbErr = fmt.Errorf("failed to connect database: %w", err)
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			dbErr = fmt.Errorf("failed to get sql.DB from gorm: %w", err)
			return
		}

		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenDbConn)
		sqlDB.SetConnMaxIdleTime(dbConfig.MaxIdleDbConn)
		sqlDB.SetConnMaxLifetime(dbConfig.MaxDbLifeTime)

		dbInstance = &PostgresDatabase{DB: db, Config: dbConfig}
	})

	if dbErr != nil {
		return nil, dbErr
	}

	return dbInstance, nil
}

func (pgx *PostgresDatabase) GetGormDB() *gorm.DB {
	return pgx.DB
}

func (pgx *PostgresDatabase) WithTx(tx *gorm.DB) Database {
	return &PostgresDatabase{DB: tx, Config: pgx.Config}
}

func (pgx *PostgresDatabase) GetDBConfig() Config {
	return *pgx.Config
}
