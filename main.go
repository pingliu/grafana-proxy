package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-cas/cas"
	"github.com/go-chi/chi"
	"github.com/golang/glog"
)

var (
	casURL      string
	grafanaAddr string
	serviceAddr string
)

func init() {
	flag.StringVar(&casURL, "cas-url", "", "cas url")
	flag.StringVar(&grafanaAddr, "grafana-addr", "localhost:3000", "grafana addr")
	flag.StringVar(&serviceAddr, "service-addr", ":8080", "grafana proxy addr")
	flag.Parse()
}

func main() {
	url, _ := url.Parse(casURL)
	client := cas.NewClient(&cas.Options{URL: url})

	root := chi.NewRouter()
	root.Use(client.Handler)
	root.HandleFunc("/*", handle)

	server := &http.Server{
		Addr:    serviceAddr,
		Handler: client.Handle(root),
	}
	if err := server.ListenAndServe(); err != nil {
		glog.Fatal(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	user := strings.Split(cas.Username(r), "@")[0]
	r.Header.Add("X-WEBAUTH-USER", user)
	r.Host, r.URL.Host = grafanaAddr, grafanaAddr
	r.RequestURI, r.URL.Scheme = "", "http"

	resp, err := (&http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}).Do(r)

	if err != nil {
		glog.Fatal(err)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Fatal(err)
	}
	defer resp.Body.Close()

	header := w.Header()
	for key, value := range resp.Header {
		for _, item := range value {
			header.Add(key, item)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(w, bytes.NewReader(b)); err != nil {
		glog.Fatal(err)
	}
}
