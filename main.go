// service2.go
package main

import (
	"io"
	"log"
	"net/http"
)

const service1BaseURL = "http://localhost:8081"

type Forwarder interface {
	ForwardRequest(w http.ResponseWriter, r *http.Request)
}

func CUIForwarderHandler(f Forwarder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.ForwardRequest(w, r)
	}
}

func main() {
	c := CUIForwarder{}
	http.HandleFunc("/", CUIForwarderHandler(&c))
	log.Println("Service 2 running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type CUIForwarder struct {
	c *http.Client
}

func (f *CUIForwarder) ForwardRequest(w http.ResponseWriter, r *http.Request) {
	targetURL := service1BaseURL + r.URL.Path

	req, err := http.NewRequest(r.Method, targetURL+"?"+r.URL.RawQuery, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}
