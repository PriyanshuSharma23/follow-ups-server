package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"sync"
	"time"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/jsonlogger"
	_ "github.com/lib/pq"
)

var (
	version   string
	buildTime string
)

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxIdleTime  string
		maxIdleConns int
		maxOpenConns int
	}
}

type application struct {
	config config
	logger *jsonlogger.Logger
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "default port of server")
	flag.StringVar(&cfg.env, "env", "development", "server environment: (development | production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgresSQL DSN")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgresSQL max idle connecitons")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgresSQL max open connecitons")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgresSQL max connection idle time")

	flag.Parse()

	logger := jsonlogger.NewLogger(os.Stdout, jsonlogger.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	logger.PrintInfo("database pool established", nil)

	app := &application{
		config: cfg,
		logger: logger,
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	d, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(d)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
