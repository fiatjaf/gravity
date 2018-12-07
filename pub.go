package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

func wrapCreate(note PubNote) (create PubCreate) {
	return PubCreate{
		PubBase: PubBase{
			Type: "Create",
			Id:   s.ServiceURL + "/pub/create/" + note.RawId,
		},
		Actor:  s.ServiceURL + "/pub/" + note.Owner,
		Object: note,
	}
}

type PubBase struct {
	Context []string `json:"@context,omitempty"`
	Id      string   `json:"id"`
	Type    string   `json:"type"`
}

type PubActor struct {
	PubBase

	Name                      string     `json:"name"`
	PreferredUsername         string     `json:"preferredUsername"`
	ManuallyApprovesFollowers bool       `json:"manuallyApprovesFollowers"`
	Image                     ActorImage `json:"image,omitempty"`
	Icon                      ActorImage `json:"icon,omitempty"`
	Summary                   string     `json:"summary,omitempty"`
	URL                       string     `json:"url"`
	Inbox                     string     `json:"inbox"`
	Outbox                    string     `json:"outbox"`
	Followers                 string     `json:"followers"`

	PublicKey ActorKey `json:"publicKey"`
}

type ActorImage struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

type ActorKey struct {
	Id           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPEM string `json:"publicKeyPem"`
}

type PubAccept struct {
	PubBase

	Object interface{} `json:"object"`
}

type PubOrderedCollection struct {
	PubBase

	TotalItems int                      `json:"totalItems"`
	First      PubOrderedCollectionPage `json:"first"`
}

type PubOrderedCollectionPage struct {
	PubBase

	TotalItems   int         `json:"totalItems"`
	PartOf       string      `json:"partOf"`
	OrderedItems interface{} `json:"orderedItems"`
}

type PubFollow struct {
	PubBase

	Actor  string `json:"actor"`
	Object string `json:"object"`
}

type PubCreate struct {
	PubBase

	Actor  string      `json:"actor"`
	Object interface{} `json:"object"`
}

type PubNote struct {
	PubBase

	Published    string `json:"published" db:"published"`
	AttributedTo string `json:"attributedTo" db:"attributedTo"`
	Content      string `json:"content" db:"content"`
	To           string `json:"to" db:"to"`

	RawId string `json:"-" db:"raw_id"`
	Owner string `json:"-" db:"owner"`
}

func pubUserActor(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]

	var exists int
	err := pg.Get(&exists, `SELECT count(*) FROM users WHERE name = $1`, owner)
	if err != nil {
		http.Error(w, "User not found", 404)
		return
	}

	image := ActorImage{
		Type: "Image",
		URL:  s.ServiceURL + "/icon.svg",
	}

	actor := PubActor{
		PubBase: PubBase{
			Context: []string{
				"https://www.w3.org/ns/activitystreams",
				"https://w3id.org/security/v1",
			},
			Id:   s.ServiceURL + "/pub/" + owner,
			Type: "Person",
		},

		Name:                      owner,
		PreferredUsername:         owner,
		Followers:                 s.ServiceURL + "/pub/" + owner + "/followers",
		ManuallyApprovesFollowers: false,
		Image:  image,
		Icon:   image,
		URL:    s.ServiceURL + "/" + owner,
		Inbox:  s.ServiceURL + "/pub",
		Outbox: s.ServiceURL + "/pub/" + owner + "/outbox",

		PublicKey: ActorKey{
			Id:           s.ServiceURL + "/pub/" + owner + "#main-key",
			Owner:        s.ServiceURL + "/pub/" + owner,
			PublicKeyPEM: s.PublicKeyPEM,
		},
	}

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(actor)
}

func pubUserFollowers(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]

	followers := make([]string, 0)
	pg.Select(&followers, `
        SELECT follower
        FROM pub_user_followers
        WHERE target = $1
    `, owner)

	context := []string{
		"https://www.w3.org/ns/activitystreams",
		"https://pleroma.site/schemas/litepub-0.1.jsonld",
	}

	page := PubOrderedCollectionPage{
		PubBase: PubBase{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/" + owner + "/followers?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/" + owner + "/followers",
		TotalItems:   len(followers),
		OrderedItems: followers,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("page") != "" {
		page.PubBase.Context = context
		json.NewEncoder(w).Encode(page)
	} else {
		collection := PubOrderedCollection{
			PubBase: PubBase{
				Context: context,
				Type:    "OrderedCollection",
				Id:      s.ServiceURL + "/pub/" + owner + "/followers",
			},
			First:      page,
			TotalItems: len(followers),
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubOutbox(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]

	var notes []PubNote
	err := pg.Select(&notes, `
        SELECT *, $1 || '/pub/' || owner AS "attributedTo" FROM pub_outbox
        WHERE owner = $2
    `, s.ServiceURL, owner)
	if err == sql.ErrNoRows {
		notes = make([]PubNote, 0)
	} else if err != nil {
		log.Warn().Err(err).Str("owner", owner).Msg("error fetching stuff from database")
		http.Error(w, "Failed to fetch activities.", 500)
		return
	}

	context := []string{
		"https://www.w3.org/ns/activitystreams",
		"https://pleroma.site/schemas/litepub-0.1.jsonld",
	}

	creates := make([]PubCreate, len(notes))
	for i, note := range notes {
		note.AttributedTo = s.ServiceURL + "/pub/" + note.Owner
		note.PubBase.Id = s.ServiceURL + "/pub/note/" + note.RawId
		creates[i] = wrapCreate(note)
	}

	page := PubOrderedCollectionPage{
		PubBase: PubBase{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/" + owner + "/followers?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/" + owner + "/followers",
		TotalItems:   len(creates),
		OrderedItems: creates,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("max_id") != "" {
		page.PubBase.Context = context
		json.NewEncoder(w).Encode(page)
	} else {
		collection := PubOrderedCollection{
			PubBase: PubBase{
				Context: context,
				Type:    "OrderedCollection",
				Id:      s.ServiceURL + "/pub/" + owner + "/outbox",
			},
			First:      page,
			TotalItems: page.TotalItems,
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubNote(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var note PubNote
	err := pg.Get(&note, `
        SELECT *, $1 || '/pub/' || owner AS "attributedTo" FROM pub_outbox
        WHERE id  = $2
    `, s.ServiceURL, id)
	if err != nil {
		http.Error(w, "Note not found", 404)
		return
	}

	note.PubBase.Context = []string{
		"https://www.w3.org/ns/activitystreams",
		"https://pleroma.site/schemas/litepub-0.1.jsonld",
	}

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(note)
}

func pubCreate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var note PubNote
	err := pg.Get(&note, `
        SELECT *, $1 || '/pub/' || owner AS "attributedTo" FROM pub_outbox
        WHERE id  = $2
    `, s.ServiceURL, id)
	if err != nil {
		http.Error(w, "Note not found", 404)
		return
	}

	note.PubBase.Context = []string{
		"https://www.w3.org/ns/activitystreams",
		"https://pleroma.site/schemas/litepub-0.1.jsonld",
	}

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(note)
}

func pubInbox(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)

	var err error

	switch gjson.GetBytes(b, "type").String() {
	case "Follow":
		actor := gjson.GetBytes(b, "actor").String()
		object := gjson.GetBytes(b, "object").String()
		parts := strings.Split(object, "/")
		user_target := parts[len(parts)-1]

		_, err = pg.Exec(`
            INSERT INTO pub_user_followers (follower, target)
            VALUES ($1, $2)
        `, actor, user_target)

		if err != nil && err != sql.ErrNoRows {
			log.Warn().Err(err).Str("actor", actor).Str("object", object).
				Msg("error saving Follow")
			http.Error(w, "Failed to accept Follow.", 500)
			return
		}
		break
	}

	w.WriteHeader(200)
}
