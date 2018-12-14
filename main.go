package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fiatjaf/litepub"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

type Settings struct {
	ServiceName   string `envconfig:"SERVICE_NAME" required:"true"`
	ServiceURL    string `envconfig:"SERVICE_URL" required:"true"`
	Port          string `envconfig:"PORT" required:"true"`
	PostgresURL   string `envconfig:"DATABASE_URL" required:"true"`
	IconSVG       string `envconfig:"ICON"`
	PrivateKeyPEM string `envconfig:"PRIVATE_KEY"`
	PrivateKey    *rsa.PrivateKey
	PublicKey     rsa.PublicKey
	PublicKeyPEM  string
}

var err error
var s Settings
var r *mux.Router
var pub litepub.LitePub
var pg *sqlx.DB
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

func main() {
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	// key stuff (needed for the activitypub integration)
	if s.PrivateKeyPEM != "" {
		s.PrivateKeyPEM = strings.Replace(s.PrivateKeyPEM, "$$", "\n", -1)
		decodedskpem, _ := pem.Decode([]byte(s.PrivateKeyPEM))

		sk, err := x509.ParsePKCS1PrivateKey(decodedskpem.Bytes)
		if err != nil {
			log.Fatal().Err(err).Msg("couldn't process private key pem.")
		}

		s.PrivateKey = sk
		s.PublicKey = sk.PublicKey

		key, err := x509.MarshalPKIXPublicKey(&sk.PublicKey)
		if err != nil {
			log.Fatal().Err(err).Msg("couldn't marshal public key to pem.")
		}
		s.PublicKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: key,
		}))
	}

	pub = litepub.LitePub{
		PrivateKey: s.PrivateKey,
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
	r.Path("/icon.svg").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			fmt.Fprint(w, s.IconSVG)
			return
		})

	r.Path("/pub").HandlerFunc(pubInbox)
	r.Path("/pub/user/{owner:[\\d\\w-]+}").Methods("GET").HandlerFunc(pubUserActor)
	r.Path("/pub/user/{owner:[\\d\\w-]+}/followers").Methods("GET").HandlerFunc(pubUserFollowers)
	r.Path("/pub/user/{owner:[\\d\\w-]+}/outbox").Methods("GET").HandlerFunc(pubOutbox)
	r.Path("/pub/create/{id}").Methods("GET").HandlerFunc(pubCreate)
	r.Path("/pub/note/{id}").Methods("GET").HandlerFunc(pubNote)
	r.Path("/.well-known/webfinger").HandlerFunc(webfinger)

	r.Path("/{owner}").Methods("POST").HandlerFunc(registerUser)
	r.Path("/{owner}/").Methods("POST").HandlerFunc(registerUser)

	r.Path("/{owner}").Methods("PATCH").HandlerFunc(updateUser)
	r.Path("/{owner}/").Methods("PATCH").HandlerFunc(updateUser)

	r.Path("/{owner}/{name}").Methods("PUT").HandlerFunc(setName)
	r.Path("/{owner}/{name}/").Methods("PUT").HandlerFunc(setName)

	r.Path("/{owner}/{name}").Methods("PATCH").HandlerFunc(updateName)
	r.Path("/{owner}/{name}/").Methods("PATCH").HandlerFunc(updateName)

	r.Path("/{owner}/{name}").Methods("DELETE").HandlerFunc(delName)
	r.Path("/{owner}/{name}/").Methods("DELETE").HandlerFunc(delName)

	r.Path("/").Methods("GET").Queries("cid", "").
		HandlerFunc(switchHTMLJSON(queryCIDs))
	r.Path("/{owner:[\\d\\w-]+}").Methods("GET").Queries("cid", "").
		HandlerFunc(switchHTMLJSON(queryCIDs))
	r.Path("/{owner:[\\d\\w-]+}/").Methods("GET").Queries("cid", "").
		HandlerFunc(switchHTMLJSON(queryCIDs))

	r.Path("/{owner:[\\d\\w-]+}").Methods("GET").
		HandlerFunc(switchHTMLJSON(getUser))

	r.Path("/").Methods("GET").
		HandlerFunc(switchHTMLJSON(listNames))
	r.Path("/{owner:[\\d\\w-]+}/").Methods("GET").
		HandlerFunc(switchHTMLJSON(listNames))

	r.Path("/{owner:[\\d\\w-]+}/{name:[\\d\\w-.]+}").Methods("GET").
		HandlerFunc(switchHTMLJSON(getName))
	r.Path("/{owner:[\\d\\w-]+}/{name:[\\d\\w-.]+}/").Methods("GET").
		HandlerFunc(switchHTMLJSON(getName))

	r.Path("/r/{owner}/{name}").Methods("GET").HandlerFunc(redirectName)
	r.Path("/r/{owner}/{name}/").Methods("GET").HandlerFunc(redirectName)

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

func switchHTMLJSON(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.ServeFile(w, r, "static/index.html")
		} else {
			w.Header().Set("Content-Type", "application/json")
			next(w, r)
		}
	}
}
