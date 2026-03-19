package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
)

func main() {
	seasonUrl := "https://www.thesportsdb.com/season/4391-nfl/2023"
	resp, err := http.Get(seasonUrl)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	
	// Find the poster image.
	// Looking for an image URL containing "poster".
	reImg := regexp.MustCompile(`(https://[^\s'"]+poster[^\s'"]+\.jpg)`)
	imgMatches := reImg.FindAllStringSubmatch(body, -1)
	
	fmt.Println("Found poster links on 2023 season page:")
	visited := make(map[string]bool)
	for _, m := range imgMatches {
		link := m[1]
		if !visited[link] {
			fmt.Printf("Link: %s\n", link)
			visited[link] = true
		}
	}
}
