package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ChaosNyaruko/ondict/sources"
)

type proxy struct {
	timeout *time.Timer
}

func (s *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.timeout.Stop() {
		select {
		case t := <-s.timeout.C: // try to drain from the channel
			log.Printf("drained from timer: %v", t)
		default:
		}
	}
	if *idleTimeout > 0 {
		s.timeout.Reset(*idleTimeout)
	}
	log.Printf("query HTTP path: %v", r.URL.Path)
	if strings.HasSuffix(r.URL.Path, "/dict") {
		q := r.URL.Query()
		word := q.Get("query")
		e := q.Get("engine")
		f := q.Get("format")
		log.Printf("query dict: %v, engine: %v, format: %v", word, e, f)

		res := query(word, e, f)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(res))
		if f == "html" {
			w.Write([]byte("<style>" + sources.GlobalDict.CSS() + "</style>"))
		}
		// w.Write([]byte("<style>" + odecss + "</style>"))
		// w.Write([]byte(fmt.Sprintf(`<link ref="stylesheet" type="text/css", href=/d/static/oald9.css />`)))
		return
	}
	// if strings.HasSuffix(r.URL.Path, ".css") {
	// 	log.Printf("static info: %v", r.URL.Path)
	// 	http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
	// 	return
	// }
	http.FileServer(http.Dir(".")).ServeHTTP(w, r)
}

func ParseAddr(listen string) (network string, address string) {
	// Allow passing just -remote=auto, as a shorthand for using automatic remote
	// resolution.
	if listen == "auto" {
		return "auto", ""
	}
	if parts := strings.SplitN(listen, ";", 2); len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "tcp", listen
}
