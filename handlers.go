package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

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
	auth := r.Header.Get("Authorization")
	if auth == "" {
		log.Warn().Err(err).Msg("missing authorization header")
		http.Error(w, "Missing Authorization header.", 401)
		return
	}

	token := strings.Split(auth, " ")[1]
	user, err := acd.VerifyAuth(token)
	if err != nil {
		log.Warn().Err(err).Str("token", token).Msg("wrong authorization token")
		http.Error(w, "Wrong authorization token.", 401)
		return
	}

	owner := mux.Vars(r)["owner"]
	if owner != user {
		log.Warn().Err(err).Str("auth-as", user).Str("needed", owner).
			Msg("authorized for a different user")
		http.Error(w, "Authorized for a different user: "+user, 401)
		return
	}

	name := mux.Vars(r)["name"]

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Missing IPFS hash.", 400)
		return
	}
	cid := string(data)

	_, err = pg.Exec(`
        INSERT INTO head (owner, name, cid)
        VALUES ($1, $2, $3)
        ON CONFLICT (owner, name) DO
        UPDATE SET cid = $3, updated_at = now()
    `, owner, name, cid)

	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).Str("cid", cid).
			Msg("error updating record")
	}

	w.WriteHeader(200)
}
