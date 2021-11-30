package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-zoo/bone"
	"github.com/n0madic/fullfeed"
)

func handleFeed(w http.ResponseWriter, r *http.Request) {
	feedName := bone.GetValue(r, "feed")
	if feedName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("feed name required"))
		return
	}

	feedConfig, ok := config.Feeds[feedName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("feed not found"))
		return
	}

	var err error
	var response string

	entryIndex := bone.GetValue(r, "entry")
	if entryIndex != "" {
		sourceFeed, err := fullfeed.LoadSourceFeed(feedConfig)
		if !checkOK(err) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		index, err := strconv.Atoi(entryIndex)
		if !checkOK(err) || index > len(sourceFeed.Items)-1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("correct entry index required"))
			return
		}

		response, err = fullfeed.GetFullContent(feedConfig, sourceFeed.Items[index].Link.Href)
		if !checkOK(err) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	} else {
		feed, errors := fullfeed.GetFullFeed(feedConfig)
		for _, err := range errors {
			log.Printf("[%s] error: %s", feedName, err)

			if feed == nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error getting full feed"))
				return
			}
		}

		w.Header().Set("Content-Type", "application/xml")
		response, err = feed.ToRss()
	}

	if !checkOK(err) || response == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(response))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("feed")
	if query != "" {
		http.Redirect(w, r, "/feed/"+query, http.StatusMovedPermanently)
	} else {
		t, err := template.New("index").Parse(indexTpl)
		if checkOK(err) {
			var tpl bytes.Buffer
			err = t.Execute(&tpl, config)
			if checkOK(err) {
				w.Write(tpl.Bytes())
			}
		}
	}
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	w.Write(rssicon)
}
