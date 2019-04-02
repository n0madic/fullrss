package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	readability "github.com/go-shiori/go-readability"
	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	pool "gopkg.in/go-playground/pool.v3"
)

type feed struct {
	BaseHref    string `yaml:"base_href"`
	Description string `yaml:"description"`
	Filters     struct {
		Selectors []string `yaml:"selectors"`
		Text      []string `yaml:"text"`
		Titles    []string `yaml:"titles"`
	}
	MaxWorkers  uint   `yaml:"max_workers"`
	Method      string `yaml:"method"`
	MethodQuery string `yaml:"method_query"`
	URL         string `yaml:"url"`
}

func getURL(url string) string {
	if !noURLCache {
		if content, ok := urlCache.Get(url); ok {
			return content.(string)
		}
	}
	client := http.Client{
		Timeout: time.Duration(30 * time.Second),
	}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "FullRSS proxy")
	res, err := client.Do(req)
	if check(err) {
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			log.Printf("status code for url %s : %s", url, res.Status)
		} else {
			utf8, err := charset.NewReader(res.Body, res.Header.Get("Content-Type"))
			bodyBytes, err := ioutil.ReadAll(utf8)
			if check(err) {
				bodyString := string(bodyBytes)
				if !noURLCache {
					urlCache.Add(url, bodyString)
				}
				return bodyString
			}
		}
	}
	return ""
}

func getFullContent(feedConfig feed, feedItem *gofeed.Item) pool.WorkFunc {
	return func(wu pool.WorkUnit) (interface{}, error) {
		fullContent := feedItem.Description
		content := strings.NewReader(getURL(feedItem.Link))
		if wu.IsCancelled() {
			return nil, nil
		}
		if content.Len() > 0 {
			baseHref := feedItem.Link
			if feedConfig.BaseHref != "" {
				baseHref = feedConfig.BaseHref
			}
			baseURL, err := url.Parse(baseHref)
			check(err)
			switch feedConfig.Method {
			case "query":
				doc, err := goquery.NewDocumentFromReader(content)
				if check(err) {
					fullContent, _ = doc.Find(feedConfig.MethodQuery).Html()
				}
			case "xpath":
				doc, err := htmlquery.Parse(content)
				if check(err) {
					list := htmlquery.Find(doc, feedConfig.MethodQuery)
					if len(list) > 0 {
						var b bytes.Buffer
						err = html.Render(&b, list[0])
						check(err)
						fullContent = b.String()
					}
				}
			default:
				doc, err := readability.FromReader(content, baseURL.String())
				if check(err) {
					fullContent = doc.Content
				}
			}
			if fullContent == "" || utf8.RuneCountInString(fullContent) < utf8.RuneCountInString(feedItem.Description) {
				fullContent = feedItem.Description
			} else {
				doc, err := goquery.NewDocumentFromReader(strings.NewReader(fullContent))
				if check(err) {
					if len(feedConfig.Filters.Selectors) > 0 {
						doc.Find(strings.Join(feedConfig.Filters.Selectors, ", ")).Remove()
					}
					if len(feedConfig.Filters.Text) > 0 {
						var searchText []string
						for _, text := range feedConfig.Filters.Text {
							searchText = append(searchText, fmt.Sprintf("div:contains('%s')", text))
						}
						doc.Find(strings.Join(searchText, ", ")).Remove()
					}
					makeAllLinksAbsolute(baseURL, doc)
					fullContent, err = doc.Html()
					if !check(err) {
						fullContent = feedItem.Description
					}
				}
			}
		}
		return &feeds.Item{
			Title: feedItem.Title,
			Link:  &feeds.Link{Href: feedItem.Link},
			Author: &feeds.Author{
				Name:  checkStructField(feedItem.Author, "Name", ""),
				Email: checkStructField(feedItem.Author, "Email", "")},
			Created:     *feedItem.PublishedParsed,
			Description: fullContent,
		}, nil
	}
}

func getFullFeed(feed string, entry string) string {
	var rss string
	fp := gofeed.NewParser()
	sourceFeed, err := fp.ParseURL(config.Feeds[feed].URL)
	if check(err) {
		maxWorkers := config.Feeds[feed].MaxWorkers
		if maxWorkers == 0 {
			maxWorkers = 10
		}
		p := pool.NewLimited(maxWorkers)
		defer p.Close()
		batch := p.Batch()
		if entry == "" {
			for i := 0; i < len(sourceFeed.Items); i++ {
				if !stringIsFiltered(sourceFeed.Items[i].Title, config.Feeds[feed].Filters.Titles) {
					batch.Queue(getFullContent(config.Feeds[feed], sourceFeed.Items[i]))
				}
			}
		} else {
			index, err := strconv.ParseInt(entry, 10, 0)
			if check(err) {
				batch.Queue(getFullContent(config.Feeds[feed], sourceFeed.Items[index]))
			}
		}
		batch.QueueComplete()
		fullFeed = &feeds.Feed{
			Title:       sourceFeed.Title,
			Link:        &feeds.Link{Href: sourceFeed.Link},
			Description: sourceFeed.Description,
		}
		for item := range batch.Results() {
			if check(item.Error()) {
				fullFeed.Add(item.Value().(*feeds.Item))
			}
		}
		sort.Slice(fullFeed.Items, func(i, j int) bool {
			return fullFeed.Items[j].Created.Before(fullFeed.Items[i].Created)
		})
		rss, err = fullFeed.ToRss()
		check(err)
	}
	return rss
}

func urlCacheWarming() {
	var wg sync.WaitGroup
	defer timeTrack(time.Now(), "URL cache warming")
	log.Println("Start warm up...")
	for feed := range config.Feeds {
		wg.Add(1)
		go func(feed string) {
			defer wg.Done()
			getFullFeed(feed, "")
			log.Println(fmt.Sprintf("Cache warm up for %s completed", feed))
		}(feed)
	}
	wg.Wait()
	log.Println("Number of cached objects:", urlCache.Len())
}
