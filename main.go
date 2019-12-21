package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-zoo/bone"
	"github.com/n0madic/fullfeed"
	yaml "gopkg.in/yaml.v2"
)

var (
	bindHost string
	config   = struct {
		Feeds map[string]fullfeed.Config
	}{}
	configYAML    string
	noURLCache    bool
	noWarmupCache bool
)

func buildMux() http.Handler {
	mux := bone.New()
	mux.Get("/", http.HandlerFunc(handleRoot))
	mux.Get("/favicon.ico", http.HandlerFunc(handleFavicon))
	mux.Get("/feed/:feed", http.HandlerFunc(handleFeed))
	mux.Get("/entry/:feed/:entry", http.HandlerFunc(handleFeed))
	return mux
}

func init() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	flag.StringVar(&configYAML, "config", "fullrss.yaml", "Config file")
	flag.StringVar(&bindHost, "bind", ":"+port, "Bind address")
	flag.BoolVar(&noURLCache, "nocache", false, "Disable URL cache")
	flag.BoolVar(&noWarmupCache, "nowarm", false, "No warm up URL cache")
}

func main() {
	flag.Parse()
	yamlFile, err := ioutil.ReadFile(configYAML)
	if checkOK(err) {
		err = yaml.Unmarshal(yamlFile, &config)
		if checkOK(err) {
			if !noURLCache {
				err = fullfeed.InitContentCache(1000)
				if checkOK(err) {
					if !noWarmupCache {
						go urlCacheWarming()
					}
				}
			}
			srv := &http.Server{
				Handler:      buildMux(),
				Addr:         bindHost,
				WriteTimeout: 60 * time.Second,
				ReadTimeout:  60 * time.Second,
			}
			log.Fatal(srv.ListenAndServe())
		}
	}
}

func urlCacheWarming() {
	var wg sync.WaitGroup
	defer timeTrack(time.Now(), "URL cache warming")
	log.Println("Start warm up...")
	for feed := range config.Feeds {
		wg.Add(1)
		go func(feed string) {
			defer wg.Done()
			_, errors := fullfeed.GetFullFeed(config.Feeds[feed])
			for _, err := range errors {
				log.Printf("[%s] error: %s", feed, err)
			}
			log.Println(fmt.Sprintf("Cache warm up for %s completed", feed))
		}(feed)
	}
	wg.Wait()
	log.Println("Number of cached objects:", fullfeed.ContentCacheLength())
}
