package main

import (
	"database/sql"
	"errors"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type database struct {
	*bun.DB
}

func newDB() *database {
	var connector *pgdriver.Connector

	if !strings.EqualFold(os.Getenv("DATABASE_URL"), "") {
		dsn := os.Getenv("DATABASE_URL")
		connector = pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	} else {
		connector = pgdriver.NewConnector(
			pgdriver.WithAddr(os.Getenv("DB_ADDR")),
			pgdriver.WithUser(os.Getenv("DB_USER")),
			pgdriver.WithPassword(os.Getenv("DB_PASS")),
			pgdriver.WithDatabase(os.Getenv("DB_NAME")),
		)
	}

	sqldb := sql.OpenDB(connector)
	db := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())
	if db == nil {
		log.Fatal().Err(errors.New("failed to connect db"))
	}

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true), // log everything
	))

	log.Info().Msg("db connected")
	return &database{
		db,
	}
}
