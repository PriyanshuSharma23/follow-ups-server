package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PriyanshuSharma23/follow-ups-server/internals/data"
	"github.com/PriyanshuSharma23/follow-ups-server/internals/jsonlogger"
	"github.com/PriyanshuSharma23/follow-ups-server/internals/mailer"
	_ "github.com/lib/pq"
)

var (
	version   string
	buildTime string
)

type config struct {
	port int
	env  string
	cors struct{ trustedOrigins []string }
	db   struct {
		dsn          string
		maxIdleTime  string
		maxIdleConns int
		maxOpenConns int
	}
	smtp struct {
		host     string
		username string
		password string
		sender   string
		port     int
		retries  int
	}
	limter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

type application struct {
	config config
	logger *jsonlogger.Logger
	models data.Models
	mailer mailer.Mailer
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

	flag.Float64Var(&cfg.limter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second.")
	flag.IntVar(&cfg.limter.burst, "limiter-burst", 4, "Rate limiter maximum burst.")
	flag.BoolVar(&cfg.limter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "7beb0df3023aa3", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "1e59f8334837b1", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "FollowUps <inbox.priyanshu@gmail.com>", "SMTP sender")
	flag.IntVar(&cfg.smtp.retries, "smtp-retires", 3, "SMTP number of retries for failed email delivery")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(s string) error {
		cfg.cors.trustedOrigins = strings.Fields(s)
		return nil
	})

	displayVersion := flag.Bool("version", false, "Display current version of the application")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:   \t%s\n", version)
		fmt.Printf("Build Time:\t%s\n", buildTime)
		os.Exit(0)
	}

	logger := jsonlogger.NewLogger(os.Stdout, jsonlogger.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	logger.PrintInfo("database pool established", nil)

	m, err := mailer.New(
		cfg.smtp.host,
		cfg.smtp.port,
		cfg.smtp.username,
		cfg.smtp.password,
		cfg.smtp.sender,
		cfg.smtp.retries,
	)

	if err != nil {
		panic(err)
	}

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: m,
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
