package main

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

var CONTEXT = []string{
	"https://www.w3.org/ns/activitystreams",
	"https://w3id.org/security/v1",
	"https://pleroma.site/schemas/litepub-0.1.jsonld",
}

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

func theirInbox(theirId string) (string, error) {
	r, _ := http.NewRequest("GET", theirId, nil)
	r.Header.Set("Accept", "application/activity+json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return "", err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	inbox := gjson.GetBytes(b, "inbox").String()
	if inbox == "" {
		return "", errors.New("didn't find .inbox property on " + string(b)[:100] + "...")
	}

	return inbox, nil
}

func sendSigned(url string, data interface{}) (*http.Response, error) {
	body := &bytes.Buffer{}
	json.NewEncoder(body).Encode(data)
	r, _ := http.NewRequest("POST", url, body)

	date := time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	b, _ := ioutil.ReadAll(r.Body)
	digest := sha256.Sum256(b)
	signed := fmt.Sprintf("(request-target): post %s\nhost: %s\ndate: %s",
		r.URL.Path, r.Host, date)
	log.Print(signed)
	hashed := sha256.Sum256([]byte(signed))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.PrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	sigheader := fmt.Sprintf(
		`keyId="%s",headers="(request-target) host date",signature="%s",algorithm="rsa-sha256"`,
		s.ServiceURL+"/pub/key", base64.StdEncoding.EncodeToString(signature))

	r.Header.Set("Content-Type", "application/activity+json")
	r.Header.Set("Digest", fmt.Sprintf("SHA2-256=%x", digest))
	r.Header.Set("Signature", sigheader)
	r.Header.Set("Date", date)
	r.Header.Set("Host", r.Host)

	return http.DefaultClient.Do(r)
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
			Context: []string{},
			Id:      s.ServiceURL + "/pub/" + owner,
			Type:    "Person",
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
		page.PubBase.Context = CONTEXT
		json.NewEncoder(w).Encode(page)
	} else {
		collection := PubOrderedCollection{
			PubBase: PubBase{
				Context: CONTEXT,
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
		page.PubBase.Context = CONTEXT
		json.NewEncoder(w).Encode(page)
	} else {
		collection := PubOrderedCollection{
			PubBase: PubBase{
				Context: CONTEXT,
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

	j := gjson.ParseBytes(b)
	typ := j.Get("type").String()
	switch typ {
	case "Follow":
		actor := j.Get("actor").String()
		object := j.Get("object").String()
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

		url, err := theirInbox(actor)
		if err != nil {
			log.Warn().Err(err).Str("actor", actor).
				Msg("didn't found an inbox from the follower")
			http.Error(w, "Wrong Follow request.", 400)
			return
		}

		accept := PubAccept{
			PubBase: PubBase{
				Context: CONTEXT,
				Type:    "Accept",
				Id:      s.ServiceURL + "/pub/accept/" + actor + "/" + user_target,
			},
			Object: object,
		}
		resp, err := sendSigned(url, accept)
		b, _ := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Warn().Err(err).Str("body", string(b)).
				Msg("failed to send Accept")
			http.Error(w, "Failed to send Accept.", 503)
			return
		}
		log.Print(string(b))

		break
	case "Undo":
		switch j.Get("object.type").String() {
		case "Follow":
			actor := j.Get("object.actor").String()
			object := j.Get("object.object").String()
			parts := strings.Split(object, "/")
			user_target := parts[len(parts)-1]

			_, err = pg.Exec(`
                DELETE FROM pub_user_followers
                WHERE follower = $1 AND target = $2
            `, actor, user_target)

			if err != nil && err != sql.ErrNoRows {
				log.Warn().Err(err).Str("actor", actor).Str("object", object).
					Msg("error undoing Follow")
				http.Error(w, "Failed to accept Undo.", 500)
				return
			}
			break
		}
	case "Delete":
		actor := j.Get("actor").String()

		_, err = pg.Exec(`
                DELETE FROM pub_user_followers
                WHERE follower = $1
            `, actor)

		if err != nil && err != sql.ErrNoRows {
			log.Warn().Err(err).Str("actor", actor).Msg("error accepting Delete")
			http.Error(w, "Failed to accept Delete.", 500)
			return
		}
		break
	default:
		log.Info().Str("type", typ).Str("body", string(b)).Msg("got unexpected pub event")
	}

	w.WriteHeader(200)
}

func pubKey(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, s.PublicKeyPEM)
}

func pubDispatchNote(id, owner, name, cid string) {
	create := wrapCreate(PubNote{
		PubBase: PubBase{
			Id:   s.ServiceURL + "/pub/note/" + id,
			Type: "Note",
		},
		Published:    time.Now().Format(time.RFC3339),
		AttributedTo: s.ServiceURL + "/pub/" + owner,
		Content:      fmt.Sprintf("%s/%s: https://ipfs.io/ipfs/%s", owner, name, cid),
		To:           "https://www.w3.org/ns/activitystreams#Public",

		Owner: owner,
		RawId: id,
	})
	create.Context = CONTEXT
	log.Print(create)

	var followers []string
	err = pg.Select(&followers, `
SELECT follower FROM pub_user_followers
WHERE target = $1
    `, owner)
	if err != nil {
		log.Warn().Err(err).Str("owner", owner).Str("name", name).
			Msg("failed to fetch followers")
	}

	for _, target := range followers {
		log.Print(target)
		url, err := theirInbox(target)
		log.Print(url, " ", err)
		if err != nil {
			continue
		}

		resp, err := sendSigned(url, create)
		if err != nil {
			b, _ := ioutil.ReadAll(resp.Body)
			log.Warn().Err(err).Str("body", string(b)).
				Msg("failed to send Accept")
		}

		log.Print(resp.Request.Header)
		log.Print(resp.StatusCode)
		b, err := ioutil.ReadAll(resp.Body)
		log.Print(string(b))
	}
}
