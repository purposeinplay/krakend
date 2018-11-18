package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	h, _ := os.Hostname()
	f := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		fmt.Printf("%s new request:\n%s %s\nheaders: %+v\n\n%s\n", h, r.Method, r.URL.String(), r.Header, string(b))

		rw.Header().Add("Content-Type", "application/json")
		rw.Header().Add("X-Backend", h)
		rw.Write(b)
	})
	http.ListenAndServe(":8080", f)
}
