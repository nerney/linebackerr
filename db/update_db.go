package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

func main() {
	resp, err := http.Get("https://www.thesportsdb.com/league/4391-nfl")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	
	// Print a small sample to see the format of seasons
	re := regexp.MustCompile(`href=['"](/season/4391[^'"]+)['"]>([^<]+)</a>`)
	matches := re.FindAllStringSubmatch(string(body), 5)
	for _, m := range matches {
		fmt.Printf("Link: %s, Year: %s\n", m[1], m[2])
	}
}
