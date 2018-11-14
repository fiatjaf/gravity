package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/badoux/checkmail"
	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

func listNames(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]
	cid := strings.TrimSpace(r.URL.Query().Get("cid"))
	if strings.HasPrefix(cid, "/ipfs/") {
		cid = cid[6:]
	}

	var res interface{}
	var err error

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

	if err != nil && err != sql.ErrNoRows {
		log.Warn().Err(err).Str("owner", owner).Str("cid", cid).
			Msg("error fetching stuff from database")
		http.Error(w, "Error fetching data.", 500)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func getName(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]
	name := mux.Vars(r)["name"]

	// show specific key
	query := selectWithMatch(`WHERE owner = $1 AND name = $2 `)
	var entry Entry
	err = pg.Get(&entry, query, owner, name)

	res := &entry
	if err == sql.ErrNoRows {
		res = nil
	}

	if err != nil && err != sql.ErrNoRows {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).
			Msg("error fetching stuff from database")
		http.Error(w, "Error fetching data.", 500)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]

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

	w.WriteHeader(200)
}

func updateUser(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]

	var data map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON body.", 400)
		return
	}

	token := r.Header.Get("Token")
	err = validateJWT(token, owner, data)
	if err != nil {
		log.Warn().Err(err).Str("token", token).Msg("token data is invalid")
		http.Error(w, "Token data is invalid.", 401)
		return
	}

	setKeys := make([]string, len(data))
	setValues := make([]interface{}, len(data))
	setValues[0] = owner
	i := 0
	for k, v := range data {
		setKeys[i] = fmt.Sprintf("%s = %i", k, i+2)
		setValues[i+1] = v
		i++
	}
	_, err = pg.Exec(`
        UPDATE users SET
        `+strings.Join(setKeys, ", ")+`
        WHERE owner = $1
    `, setValues...)

	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Fields(data).
			Msg("error updating record")
		http.Error(w, "Error updating record: "+err.Error(), 500)
		return
	}

	w.WriteHeader(200)
}

func setName(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]
	name := mux.Vars(r)["name"]

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Missing CID in the request body.", 400)
		return
	}
	cid := string(data)

	token := r.Header.Get("Token")
	err = validateJWT(token, owner, map[string]interface{}{
		"owner": owner,
		"name":  name,
		"cid":   cid,
	})
	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).Str("cid", cid).
			Str("token", token).
			Msg("token data is invalid")
		http.Error(w, "Token data is invalid.", 401)
		return
	}

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

	w.WriteHeader(200)
}

func updateName(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]
	name := mux.Vars(r)["name"]

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Missing request body JSON.", 400)
		return
	}

	rnote := gjson.GetBytes(data, "note")
	if !rnote.Exists() {
		http.Error(w, "Missing \"note\" in body.", 400)
		return
	}
	note := rnote.String()

	token := r.Header.Get("Token")
	err = validateJWT(token, owner, map[string]interface{}{
		"owner": owner,
		"name":  name,
		"note":  note,
	})
	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).Str("note", note).
			Str("token", token).
			Msg("token data is invalid")
		http.Error(w, "Token data is invalid.", 401)
		return
	}

	_, err = pg.Exec(`
        UPDATE head SET note = $3, updated_at = now()
        WHERE owner = $1 AND name = $2
    `, owner, name, note)

	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).Str("note", note).
			Msg("error updating record")
		http.Error(w, "Error updating record: "+err.Error(), 500)
		return
	}

	w.WriteHeader(200)
}

func delName(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]
	name := mux.Vars(r)["name"]

	token := r.Header.Get("Token")
	err := validateJWT(token, owner, map[string]interface{}{
		"owner": owner,
		"name":  name,
	})
	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).
			Str("token", token).
			Msg("token data is invalid")
		http.Error(w, "Token data is invalid.", 401)
		return
	}

	_, err = pg.Exec(`
        DELETE FROM head
        WHERE owner = $1 AND name = $2
    `, owner, name)

	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).
			Msg("error updating record")
		http.Error(w, "Error updating record: "+err.Error(), 500)
		return
	}

	w.WriteHeader(200)
}
