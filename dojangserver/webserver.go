package main

import (
	"net/http"
)

func init() {
	http.HandleFunc("/jquery.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(jquery))
	})
	http.HandleFunc("/json3.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(json3))
	})
	http.HandleFunc("/bulma.css", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(css))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(webcontent))
	})
}
