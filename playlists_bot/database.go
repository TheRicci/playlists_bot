package main

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type database struct {
	*bun.DB
}

type User struct {
	ID            string
	Name          string
	Updated_at    *time.Time
	Created_at    *time.Time
	bun.BaseModel `bun:"playlistsDB_user"`
}

type Playlist struct {
	ID            string
	Title         string
	Description   string
	Thumbnail     string
	Updated_at    *time.Time
	Created_at    *time.Time
	Refreshed_at  *time.Time
	bun.BaseModel `bun:"playlistsDB_playlist"`
}

type Video struct {
	ID            string
	Title         string
	Description   string
	Thumbnail     string
	Updated_at    *time.Time
	Created_at    *time.Time
	bun.BaseModel `bun:"playlistsDB_video"`
}

type PlaylistVideo struct {
	playlist      string
	video         string
	bun.BaseModel `bun:"playlistsDB_playlist_video"`
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
