package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/im-karina/basics/cfg"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type DbWrapper struct {
	rdDb *sqlx.DB
	wrDb *sqlx.DB
}

func (db DbWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	wrMtx.Lock()
	defer wrMtx.Unlock()

	return db.wrDb.ExecContext(ctx, query, args...)
}

func (db DbWrapper) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.rdDb.QueryRowContext(ctx, query, args...)
}
func (db DbWrapper) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	return db.rdDb.QueryRowxContext(ctx, query, args...)
}
func (db DbWrapper) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.rdDb.QueryContext(ctx, query, args...)
}
func (db DbWrapper) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return db.rdDb.SelectContext(ctx, dest, query, args...)
}
func (db DbWrapper) Close() error {
	return errors.Join(db.wrDb.Close(), db.rdDb.Close())
}

var wrMtx sync.Mutex

func (db DbWrapper) Transaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	wrMtx.Lock()
	defer wrMtx.Unlock()

	tx, err := db.wrDb.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	err = fn(tx)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

var Db DbWrapper

func MustConnectOnce() {
	if Db.rdDb != nil {
		return
	}

	Db.rdDb = sqlx.MustConnect("sqlite3", cfg.DbConnectionString)
	Db.wrDb = sqlx.MustConnect("sqlite3", cfg.DbConnectionString)
	Db.rdDb.MustExec(`PRAGMA journal_mode=WAL;`)
	Db.wrDb.MustExec(`PRAGMA journal_mode=WAL;`)
	Db.rdDb.MustExec(`PRAGMA busy_timeout=5000;`)
	Db.wrDb.MustExec(`PRAGMA busy_timeout=5000;`)
	Db.rdDb.MustExec(`PRAGMA synchronous=NORMAL;`)
	Db.wrDb.MustExec(`PRAGMA synchronous=NORMAL;`)
	Db.rdDb.MustExec(`PRAGMA cache_size=1000000000;`)
	Db.wrDb.MustExec(`PRAGMA cache_size=1000000000;`)
	Db.rdDb.MustExec(`PRAGMA temp_store = memory;`)
	Db.wrDb.MustExec(`PRAGMA temp_store = memory;`)
	Db.rdDb.SetMaxOpenConns(runtime.NumCPU())
	Db.wrDb.SetMaxOpenConns(1)
}

var ErrMigrationFailed = errors.New("failed to run database migration")

func Migrate(_ string) error {
	MustConnectOnce()

	c := sqlite3.Config{}
	driver, err := sqlite3.WithInstance(Db.wrDb.DB, &c)
	if err != nil {
		return errors.Join(ErrMigrationFailed, err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "sqlite3", driver)
	if err != nil {
		return errors.Join(ErrMigrationFailed, err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return errors.Join(ErrMigrationFailed, err)
	}

	if cfg.IsProd {
		return nil
	}

	return DumpSchema("")
}

var ErrRollbackFailed = errors.New("failed to run database rollback")

func Rollback(_ string) error {
	MustConnectOnce()

	cfg := sqlite3.Config{}
	driver, err := sqlite3.WithInstance(Db.wrDb.DB, &cfg)
	if err != nil {
		return errors.Join(ErrRollbackFailed, err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "sqlite3", driver)
	if err != nil {
		return errors.Join(ErrRollbackFailed, err)
	}

	err = m.Steps(-1)
	if err != nil {
		return errors.Join(ErrRollbackFailed, err)
	}

	return DumpSchema("")
}

var ErrDropFailed = errors.New("failed to run database rollback")

func Drop(_ string) error {
	if cfg.IsProd {
		return errors.Join(ErrDropFailed, cfg.ErrCannotRunInProd)
	}

	MustConnectOnce()

	cfg := sqlite3.Config{}
	driver, err := sqlite3.WithInstance(Db.wrDb.DB, &cfg)
	if err != nil {
		return errors.Join(ErrRollbackFailed, err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "sqlite3", driver)
	if err != nil {
		return errors.Join(ErrRollbackFailed, err)
	}

	err = m.Drop()
	if err != nil {
		return errors.Join(ErrRollbackFailed, err)
	}

	return DumpSchema("")
}

func DumpSchema(_ string) error {
	schemaFile, err := os.Create("db/schema.sql")
	if err != nil {
		return err
	}
	cmd := exec.Command("sqlite3", cfg.DbConnectionString, ".schema")
	cmd.Stdout = schemaFile
	return cmd.Run()
}

func WalCleanup(_ string) error {
	wrMtx.Lock()
	defer wrMtx.Unlock()

	if err := Db.Close(); err != nil {
		return err
	}

	Db.rdDb = nil
	Db.wrDb = nil

	time.Sleep(1 * time.Second)
	MustConnectOnce()

	return nil
}
