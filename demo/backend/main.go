package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	melody "gopkg.in/olahol/melody.v1"
)

const (
	proxyGreeting         = `{"msg":"KrakenD WS proxy starting"}`
	proxyGreetingResponse = "OK"
)

func main() {
	h, _ := os.Hostname()
	m := melody.New()

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		if string(msg) == proxyGreeting {
			s.Write([]byte(proxyGreetingResponse))
			return
		}
		log.Println("processing", string(msg))
		req := &Request{}
		if err := json.Unmarshal(msg, req); err != nil {
			log.Println("error:", err.Error())
			m.Broadcast(msg)
			return
		}
		m.BroadcastFilter(req.Body, func(q *melody.Session) bool {
			return q.Request.URL.Path == s.Request.URL.Path
		})
	})

	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/ws-mjolnir" {
			fmt.Printf("%s new ws request:\n%s %s\nheaders: %+v\n\n", h, r.Method, r.URL.String(), r.Header)
			if err := m.HandleRequest(rw, r); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		b, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		fmt.Printf("%s new request:\n%s %s\nheaders: %+v\n\n%s\n", h, r.Method, r.URL.String(), r.Header, string(b))

		rw.Header().Add("Content-Type", "application/json")
		rw.Header().Add("X-Backend", h)
		rw.Write(b)
	})

	http.ListenAndServe(":8080", handler)
}

type Request struct {
	URL     string            `json:"url"`
	Session map[string]string `json:"session"`
	Body    []byte            `json:"body"`
}
