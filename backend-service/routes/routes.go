package routes

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Routes(*sql.DB) http.Handler {
	mux := chi.NewRouter()

	return mux
}
