package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/itchyny/gojq"
)

func respond(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func graphql(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()
	url, err := url.Parse("https://api.us-east-1.cloud.twisp.com/financial/v1/graphql")
	if err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}
	req := &http.Request{
		Method: "POST",
		URL:    url,
		Header: map[string][]string{
			"authorization":      {fmt.Sprintf("Bearer %s", os.Getenv("SLIPLANE_OPENID_TOKEN"))},
			"x-twisp-account-id": {"01eac529-86c7-4186-9e56-3f0ec2005d3a"},
		},
		Body: r.Body,
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}

	defer func() { _ = resp.Body.Close() }()
	respB, err := io.ReadAll(resp.Body)
	if err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}

	var blah any
	if err := json.NewDecoder(bytes.NewReader(respB)).Decode(&blah); err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}

	respond(w, 200, blah)
}

func webhook(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}
	log.Printf("webhook: %s\n", string(b))
	respond(w, 200, map[string]string{})
}

func jit(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}
	log.Printf("jit: %s\n", string(b))

	var request any
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&request); err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}
	query, err := gojq.Parse(`{jit_funding:.gpa_order.jit_funding}`)
	if err != nil {
		respond(w, 500, map[string]any{"err": err})
		return
	}
	iter := query.Run(request)
	xformed, ok := iter.Next()
	if !ok {
		respond(w, 500, map[string]any{"err": "unable to apply transform"})
		return
	}

	respond(w, 200, xformed)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]string{})
	})
	http.HandleFunc("/webhook", webhook)
	http.HandleFunc("/jit", jit)
	http.HandleFunc("/graphql", graphql)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Printf("err: %s\n", err.Error())
	}
}
