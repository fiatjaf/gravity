package main

import (
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/badoux/checkmail"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

type Entry struct {
	Owner string `json:"owner" db:"owner"`
	Name  string `json:"name" db:"name"`
	CID   string `json:"cid" db:"cid"`
}

func get(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.ServeFile(w, r, "static/index.html")
	} else {
		owner := mux.Vars(r)["owner"]
		name := mux.Vars(r)["name"]
		cid := r.URL.Query().Get("cid")

		var res interface{}
		var err error

		selectWithMatch := func(match string) string {
			return `SELECT owner, name, cid FROM head ` +
				match +
				`ORDER BY updated_at DESC`
		}

		if owner != "" && name != "" {
			// show specific cid
			query := selectWithMatch(`WHERE owner = $1 AND name = $2 `)
			var entry Entry
			err = pg.Get(&entry, query, owner, name)
			res = entry
			if err == sql.ErrNoRows {
				res = nil
			}
		} else {
			match := ""
			args := []interface{}{}

			if owner == "" {
				// list all cids
				if cid != "" {
					match = `WHERE cid = $1 `
					args = append(args, cid)
				}
				query := selectWithMatch(match)
				var entries []Entry
				err = pg.Select(&entries, query, args...)
				res = entries
				if len(entries) == 0 {
					res = make([]Entry, 0)
				}
			} else {
				// list all cids for owner
				match = `WHERE owner = $1 `
				args = []interface{}{owner}
				if cid != "" {
					match += `AND cid = $2 `
					args = append(args, cid)
				}
			}

			query := selectWithMatch(match)
			var entries []Entry
			err = pg.Select(&entries, query, args...)
			res = entries
			if len(entries) == 0 {
				res = make([]Entry, 0)
			}
		}

		if err != nil && err != sql.ErrNoRows {
			log.Warn().Err(err).Str("owner", owner).Str("name", name).Str("cid", cid).
				Msg("error fetching stuff from database")
			http.Error(w, "Error fetching data.", 500)
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}

func set(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]
	name := mux.Vars(r)["name"]

	if name == "" {
		// register a new user at /owner
		if err != nil {
			http.Error(w, "Missing public key.", 400)
			return
		}

		data, err := ioutil.ReadAll(r.Body)
		pk := string(data)
		email := r.Header.Get("Email")

		if err := checkmail.ValidateFormat(email); err != nil {
			log.Warn().Err(err).Str("email", email).
				Msg("invalid email address")
			http.Error(w, "Invalid email address: "+err.Error(), 400)
			return
		}

		_, err = pg.Exec(`
            INSERT INTO users (name, email, pk)
            VALUES ($1, $2, $3)
        `, owner, email, pk)

		if err != nil {
			log.Warn().Err(err).Str("owner", owner).Str("email", email).
				Msg("error creating user")
			http.Error(w, "Error creating user: "+err.Error(), 500)
			return
		}
	} else {
		// an ipfs hash so set at /owner/name
		token := r.Header.Get("Token")

		// we get a jwt we must validate
		var pemstr string
		err := pg.Get(&pemstr, "SELECT pk FROM users WHERE name = $1", owner)
		if err != nil {
			log.Warn().Err(err).Str("owner", owner).Str("name", name).
				Str("token", token).
				Msg("failed to fetch public key")
			http.Error(w, "Failed to fetch public key: "+err.Error(), 401)
			return
		}

		block, _ := pem.Decode([]byte(pemstr))
		if block == nil || block.Type != "PUBLIC KEY" {
			log.Warn().Err(err).Str("owner", owner).Str("pem", pemstr).
				Msg("failed to decode public key from database")
			http.Error(w, "Failed to decode public key from database: "+err.Error(), 401)
			return
		}

		pk, err := x509.ParsePKCS1PublicKey(block.Bytes)
		// 	pk, err := jwt.ParseRSAPublicKeyFromPEM(block.Bytes)
		if err != nil {
			log.Warn().Err(err).Str("owner", owner).Str("pem", pemstr).
				Msg("failed to parse public key from the database")
			http.Error(w,
				"Failed to parse public key from the database: "+err.Error(),
				500)
			return
		}

		t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return pk, nil
		})
		if err != nil {
			log.Warn().Err(err).Str("owner", owner).Str("token", token).
				Msg("token is not signed by the expected key")
			http.Error(w, "Token is invalid: "+err.Error(), 401)
			return
		}

		// all data should be inside the token
		var cid string
		if claims, ok := t.Claims.(jwt.MapClaims); !ok || !t.Valid {
			goto invalid
		} else {
			if claims["owner"] != owner || claims["name"] != name {
				goto invalid
			}

			if _cid, ok := claims["cid"]; ok {
				cid = _cid.(string)
			} else {
				goto invalid
			}

			goto valid
		}

	invalid:
		log.Warn().Err(err).Str("owner", owner).Str("token", token).
			Msg("token data is invalid")
		http.Error(w, "Token data is invalid.", 401)
		return

	valid:
		_, err = pg.Exec(`
            INSERT INTO head (owner, name, cid)
            VALUES ($1, $2, $3)
            ON CONFLICT (owner, name) DO
            UPDATE SET cid = $3, updated_at = now()
        `, owner, name, cid)

		if err != nil {
			log.Warn().Err(err).Str("owner", owner).Str("name", name).Str("cid", cid).
				Msg("error updating record")
			http.Error(w, "Error updating record: "+err.Error(), 500)
			return
		}
	}

	w.WriteHeader(200)
}
