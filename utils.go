package main

import (
	"log"
	"net/url"
	"reflect"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func check(err error) bool {
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func checkStructField(Iface interface{}, fieldName string, defaultValue string) string {
	ValueIface := reflect.ValueOf(Iface)
	if ValueIface.Type().Kind() != reflect.Ptr {
		ValueIface = reflect.New(reflect.TypeOf(Iface))
	}
	if ValueIface.Elem().IsValid() {
		Field := ValueIface.Elem().FieldByName(fieldName)
		if Field.IsValid() {
			return reflect.Value(Field).String()
		}
	}
	return defaultValue
}

func absoluteAttr(base *url.URL, sel *goquery.Selection, attr string) {
	if link, ok := sel.Attr(attr); link != "" && ok {
		u, err := url.Parse(link)
		if err == nil && !u.IsAbs() {
			sel.SetAttr(attr, base.ResolveReference(u).String())
		}
	}
}

func makeAllLinksAbsolute(base *url.URL, doc *goquery.Document) {
	doc.Find("a,img").Each(func(i int, sel *goquery.Selection) {
		absoluteAttr(base, sel, "src")
		absoluteAttr(base, sel, "data-src")
		absoluteAttr(base, sel, "href")
	})
}
