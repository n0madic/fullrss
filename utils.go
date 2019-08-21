package main

import (
	"log"
	"time"
)

func checkOK(err error) bool {
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
