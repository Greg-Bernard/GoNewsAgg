package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

var wg sync.WaitGroup

// Sitemapindex retrieved from the primary sitemap index of WP
type Sitemapindex struct {
	Locations []string `xml:"sitemap>loc"`
}

// News retrived from XML: Titles, Keywords, and URLs
type News struct {
	Titles    []string `xml:"url>news>title"`
	Keywords  []string `xml:"url>news>keywords"`
	Locations []string `xml:"url>loc"`
}

// NewsMap retrieved from News to build map of news articles on host site
type NewsMap struct {
	Keyword  string
	Location string
}

// NewsAggPage is basic map and title to generate webpage
type NewsAggPage struct {
	Title string
	News  map[string]NewsMap
}

// Index page, empty for now
func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1> Hello </h1>")
}

// Create Go Routines that write to channels to process requests concurrently 
func newsRoutine(c chan News, Location string) {
	defer wg.Done()
	var n News
	// URls retrieved from WP seem to have some whitespace issues, TrimSpace to remove
	Location = strings.TrimSpace(Location)
	resp, _ := http.Get(Location)
	bytes, _ := ioutil.ReadAll(resp.Body)
	xml.Unmarshal(bytes, &n)
	resp.Body.Close()
	c <- n
}

// newAggHandler loops through the XML's returned from WP and grabs relevant titles and tags
func newsAggHandler(w http.ResponseWriter, r *http.Request) {
	var s Sitemapindex
	// WP for example, can work with any 2-level XML sitemap
	resp, _ := http.Get("https://www.washingtonpost.com/news-sitemaps/index.xml")
	bytes, _ := ioutil.ReadAll(resp.Body)
	xml.Unmarshal(bytes, &s)
	newsMap := make(map[string]NewsMap)
	resp.Body.Close()
	queue := make(chan News, 500)
	for _, Location := range s.Locations {
		wg.Add(1)
		go newsRoutine(queue, Location)
	}
	// Wait for GoRoutines to finish and close channels
	wg.Wait()
	close(queue)

	// Read from channels to populate NewsMap
	for elem := range queue {
		for idx, _ := range elem.Keywords {
			newsMap[elem.Titles[idx]] = NewsMap{elem.Keywords[idx], elem.Locations[idx]}
		}
	}

	p := NewsAggPage{Title: "News Aggregator", News: newsMap}
	t, _ := template.ParseFiles("agg.html")
	//fmt.Println(err)
	t.Execute(w, p)
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/agg/", newsAggHandler)
	http.ListenAndServe(":8000", nil)
}
