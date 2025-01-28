package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/itchyny/gojq"
)

func respond(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
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
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Printf("err: %s\n", err.Error())
	}
}
