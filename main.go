package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/davyzhang/agw"
	"github.com/go-zoo/bone"
	"github.com/gorilla/feeds"
	lru "github.com/hashicorp/golang-lru"
	yaml "gopkg.in/yaml.v2"
)

var (
	config = struct {
		Feeds map[string]feed
	}{}
	fullFeed *feeds.Feed
	urlCache *lru.Cache
)

func handleFeed(w http.ResponseWriter, r *http.Request) {
	response := getFullFeed(bone.GetValue(r, "feed"), bone.GetValue(r, "entry"))
	if response == "" {
		w.WriteHeader(http.StatusInternalServerError)
	}
	agw.WriteResponse(w, response, false)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("feed")
	if query != "" {
		http.Redirect(w, r, "/feed/"+query, 301)
	} else {
		t, err := template.New("index").Parse(indexTpl)
		if check(err) {
			var tpl bytes.Buffer
			err = t.Execute(&tpl, config)
			if check(err) {
				agw.WriteResponse(w, tpl.String(), false)
			}
		}
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	agw.WriteResponse(w, rssicon, false)
}

func buildMux() http.Handler {
	mux := bone.New()
	mux.Get("/", http.HandlerFunc(handleRoot))
	mux.Get("/favicon.ico", http.HandlerFunc(handleFavicon))
	mux.Get("/feed/:feed", http.HandlerFunc(handleFeed))
	mux.Get("/entry/:feed/:entry", http.HandlerFunc(handleFeed))
	return mux
}

func main() {
	yamlFile, err := ioutil.ReadFile("fullrss.yaml")
	if check(err) {
		err = yaml.Unmarshal(yamlFile, &config)
		if check(err) {
			urlCache, err = lru.New(1000)
			go urlCacheWarming()
			if check(err) {
				if _, ok := os.LookupEnv("LAMBDA_TASK_ROOT"); ok {
					lambda.Start(func() agw.GatewayHandler {
						return func(ctx context.Context, event json.RawMessage) (interface{}, error) {
							agp := agw.NewAPIGateParser(event)
							return agw.Process(agp, buildMux()), nil
						}
					}())
				} else {
					srv := &http.Server{
						Handler:      buildMux(),
						Addr:         ":8000",
						WriteTimeout: 60 * time.Second,
						ReadTimeout:  60 * time.Second,
					}
					log.Fatal(srv.ListenAndServe())
				}
			}
		}
	}
}
