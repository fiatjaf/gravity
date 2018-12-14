package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/fiatjaf/litepub"
	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

type DBNote struct {
	Id    string `db:"id"`
	Owner string `db:"owner"`
	Name  string `db:"name"`
	SetAt string `db:"set_at"`
	CID   string `db:"cid"`
}

func pubUserActor(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]

	var exists int
	err := pg.Get(&exists, `SELECT count(*) FROM users WHERE name = $1`, owner)
	if err != nil {
		http.Error(w, "User not found", 404)
		return
	}

	image := litepub.ActorImage{
		Type: "Image",
		URL:  s.ServiceURL + "/icon.svg",
	}

	actor := litepub.Actor{
		Base: litepub.Base{
			Context: litepub.CONTEXT,
			Id:      s.ServiceURL + "/pub/user/" + owner,
			Type:    "Person",
		},

		Name:                      owner,
		PreferredUsername:         owner,
		Followers:                 s.ServiceURL + "/pub/user/" + owner + "/followers",
		ManuallyApprovesFollowers: false,
		Image:  image,
		Icon:   image,
		URL:    s.ServiceURL + "/" + owner,
		Inbox:  s.ServiceURL + "/pub",
		Outbox: s.ServiceURL + "/pub/user/" + owner + "/outbox",

		PublicKey: litepub.PublicKey{
			Id:           s.ServiceURL + "/pub/user/" + owner + "#main-key",
			Owner:        s.ServiceURL + "/pub/user/" + owner,
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

	page := litepub.OrderedCollectionPage{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + owner + "/followers?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + owner + "/followers",
		TotalItems:   len(followers),
		OrderedItems: followers,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("page") != "" {
		page.Base.Context = litepub.CONTEXT
		json.NewEncoder(w).Encode(page)
	} else {
		collection := litepub.OrderedCollection{
			Base: litepub.Base{
				Context: litepub.CONTEXT,
				Type:    "OrderedCollection",
				Id:      s.ServiceURL + "/pub/user/" + owner + "/followers",
			},
			First:      page,
			TotalItems: len(followers),
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubOutbox(w http.ResponseWriter, r *http.Request) {
	owner := mux.Vars(r)["owner"]

	var dbnotes []DBNote
	err := pg.Select(&dbnotes, `
        SELECT
            history.id::text AS id,
            owner,
            name,
            set_at,
            history.cid
        FROM history
        INNER JOIN head ON history.record_id = head.id
        WHERE owner = $1
        ORDER BY history.set_at DESC
    `, owner)
	if err == sql.ErrNoRows {
		dbnotes = make([]DBNote, 0)
	} else if err != nil && err != sql.ErrNoRows {
		log.Warn().Err(err).Str("owner", owner).Msg("error fetching stuff from database")
		http.Error(w, "Failed to fetch activities.", 500)
		return
	}

	notes := make([]litepub.Note, len(dbnotes))
	creates := make([]litepub.Create, len(dbnotes))
	for i, dbnote := range dbnotes {
		notes[i] = makeNote(dbnote)
		creates[i] = pub.WrapCreate(notes[i], s.ServiceURL+"/pub/create/"+dbnote.Id)
	}

	page := litepub.OrderedCollectionPage{
		Base: litepub.Base{
			Type: "OrderedCollectionPage",
			Id:   s.ServiceURL + "/pub/user/" + owner + "/followers?page=1",
		},
		PartOf:       s.ServiceURL + "/pub/user/" + owner + "/followers",
		TotalItems:   len(creates),
		OrderedItems: creates,
	}

	w.Header().Set("Content-Type", "application/activity+json")
	if r.URL.Query().Get("max_id") != "" {
		page.Base.Context = litepub.CONTEXT
		json.NewEncoder(w).Encode(page)
	} else {
		collection := litepub.OrderedCollection{
			Base: litepub.Base{
				Context: litepub.CONTEXT,
				Type:    "OrderedCollection",
				Id:      s.ServiceURL + "/pub/user/" + owner + "/outbox",
			},
			First:      page,
			TotalItems: page.TotalItems,
		}
		json.NewEncoder(w).Encode(collection)
	}
}

func pubNote(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	note, err := fetchNote(id)
	if err != nil {
		http.Error(w, "Note not found", 404)
		return
	}

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(note)
}

func pubCreate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	note, err := fetchNote(id)
	if err != nil {
		http.Error(w, "Note not found", 404)
		return
	}
	create := pub.WrapCreate(note, s.ServiceURL+"/pub/create/"+id)
	create.Base.Context = litepub.CONTEXT

	w.Header().Set("Content-Type", "application/activity+json")
	json.NewEncoder(w).Encode(note)
}

func fetchNote(id string) (note litepub.Note, err error) {
	var dbnote DBNote
	err = pg.Get(&dbnote, `
        SELECT
            history.id::text AS id,
            owner,
            name,
            set_at,
            history.cid
        FROM history
        INNER JOIN head ON history.record_id = head.id
        WHERE history.id = $1
        ORDER BY history.set_at DESC
    `, id)
	if err != nil {
		return
	}

	note = makeNote(dbnote)
	note.Base.Context = litepub.CONTEXT
	return
}

func makeNote(dbnote DBNote) litepub.Note {
	return litepub.Note{
		Base: litepub.Base{
			Id:   s.ServiceURL + "/pub/note/" + dbnote.Id,
			Type: "Note",
		},
		Published:    dbnote.SetAt,
		AttributedTo: s.ServiceURL + "/pub/user/" + dbnote.Owner,
		Content: fmt.Sprintf(
			"%s/%s: https://ipfs.io/ipfs/%s",
			dbnote.Owner, dbnote.Name, dbnote.CID),
		To: "https://www.w3.org/ns/activitystreams#Public",
	}
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
            ON CONFLICT (follower, target) DO NOTHING
        `, actor, user_target)

		if err != nil && err != sql.ErrNoRows {
			log.Warn().Err(err).Str("actor", actor).Str("object", object).
				Msg("error saving Follow")
			http.Error(w, "Failed to accept Follow.", 500)
			return
		}

		url, err := litepub.FetchInbox(actor)
		if err != nil {
			log.Warn().Err(err).Str("actor", actor).
				Msg("didn't found an inbox from the follower")
			http.Error(w, "Wrong Follow request.", 400)
			return
		}

		accept := litepub.Accept{
			Base: litepub.Base{
				Context: litepub.CONTEXT,
				Type:    "Accept",
				Id:      s.ServiceURL + "/pub/accept/" + actor + "/" + user_target,
			},
			Object: object,
		}
		resp, err := pub.SendSigned(
			s.ServiceURL+"/pub/user/"+actor+"#main-key", url, accept)
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

func pubDispatchNote(id, owner, name, cid string) {
	create := pub.WrapCreate(makeNote(DBNote{
		Id:    id,
		Owner: owner,
		Name:  name,
		SetAt: time.Now().Format(time.RFC3339),
		CID:   cid,
	}), s.ServiceURL+"/pub/create/"+id)
	create.Context = litepub.CONTEXT

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
		url, err := litepub.FetchInbox(target)
		log.Print(url, " ", err)
		if err != nil {
			continue
		}

		resp, err := pub.SendSigned(
			s.ServiceURL+"/pub/user/"+owner+"#main-key", url, create)
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
