package config

import (
	"backend-service/routes"
	"database/sql"
	"log"
	"net/http"
	"os"
)

var port = ":80"

type Config struct {
	db *sql.DB
}

func InitConfig() *Config {
	conn := initDB()
	return &Config{
		db: conn,
	}
}

func initDB() *sql.DB {
	conn, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	err = conn.Ping()
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func (app *Config) InitServer() {
	srv := &http.Server{
		Addr:    port,
		Handler: routes.Routes(app.db),
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
