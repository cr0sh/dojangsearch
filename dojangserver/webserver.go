package main

import (
	"fmt"
	"net/http"
)

var cachedWebContent string = fmt.Sprintf(webcontent, func() string {
	ret := ""
	for _, world := range serverList {
		ret += fmt.Sprintf("<option value=\"%d\">%s</option>\n", world, serverName[world])
	}
	return ret
}())

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
		w.Write([]byte(cachedWebContent))
	})
}
