package main

import (
	"net/http"
	"os"
	"time"

	"github.com/fiatjaf/accountd"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

type Settings struct {
	ServiceName string `envconfig:"SERVICE_NAME" required:"true"`
	ServiceURL  string `envconfig:"SERVICE_URL" required:"true"`
	Port        string `envconfig:"PORT" required:"true"`
	PostgresURL string `envconfig:"DATABASE_URL" required:"true"`
}

var err error
var s Settings
var r *mux.Router
var pg *sqlx.DB
var acd = accountd.NewClient()
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

func main() {
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()

	// postgres connection
	pg, err = sqlx.Connect("postgres", s.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to postgres")
	}

	// define routes
	r = mux.NewRouter()
	r.Path("/favicon.ico").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./public/icon.png")
			return
		})
	r.Path("/{owner}/{name}").Methods("POST").HandlerFunc(set)
	r.Path("/{owner}/{name}/").Methods("POST").HandlerFunc(set)
	r.Path("/").Methods("GET").HandlerFunc(get)
	r.Path("/{owner:[\\d\\w-]+}").Methods("GET").HandlerFunc(get)
	r.Path("/{owner:[\\d\\w-]+}/").Methods("GET").HandlerFunc(get)
	r.Path("/{owner:[\\d\\w-]+}/{name:[\\d\\w-]+}").Methods("GET").HandlerFunc(get)
	r.Path("/{owner:[\\d\\w-]+}/{name:[\\d\\w-]+}/").Methods("GET").HandlerFunc(get)
	r.PathPrefix("/").Methods("GET").Handler(http.FileServer(http.Dir("./static")))

	// start the server
	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:" + s.Port,
		WriteTimeout: 25 * time.Second,
		ReadTimeout:  25 * time.Second,
	}
	log.Info().Str("port", s.Port).Msg("listening.")
	srv.ListenAndServe()
}
