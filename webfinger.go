package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type WebfingerResponse struct {
	Subject string          `json:"subject"`
	Links   []WebfingerLink `json:"links"`
}

type WebfingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type"`
	Href string `json:"href"`
}

func webfinger(w http.ResponseWriter, r *http.Request) {
	rsc := r.URL.Query().Get("resource")
	parts := strings.Split(rsc, "acct:")
	if len(parts) != 2 {
		http.Error(w, "Wrong Webfinger resource query.", 400)
		return
	}

	account := parts[1]
	parts = strings.Split(account, "@")
	if len(parts) != 2 {
		http.Error(w, "Wrong Webfinger resource query.", 400)
		return
	}

	name := parts[0]
	server := parts[1]

	if !strings.HasSuffix(s.ServiceURL, server) {
		http.Error(w, "Wrong Webfinger resource query.", 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WebfingerResponse{
		Subject: rsc,
		Links: []WebfingerLink{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: s.ServiceURL + "/pub/user/" + name,
			},
		},
	})
}
