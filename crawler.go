package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/jackdanger/collectlinks"
)

const usage string = `
-----------------------
Usage:
crawler <url>
-----------------------
`

var totalUrls = 1000

func filterQueue(queue chan string, filteredQueue chan string) {
	visited := make(map[string]bool)
	for uri := range queue {
		if !visited[uri] {
			visited[uri] = true
			filteredQueue <- uri
		}
	}
}

func enqueue(uri string, queue chan string) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := http.Client{Transport: transport}

	resp, err := client.Get(uri)
	if err != nil {
		fmt.Println("Error in fetching ", uri)
		return
	}
	defer resp.Body.Close()

	fmt.Println("fetched url: ", uri)
	totalUrls--

	if totalUrls == 0 {
		queue <- "done"
	} else {
		links := collectlinks.All(resp.Body)
		for _, link := range links {
			absolute := fixURL(link, uri)
			if uri != "" && absolute != "" {
				go func() { queue <- absolute }()
			}
		}
	}
}

func fixURL(href, base string) string {
	parsedURL, err := url.Parse(href)
	if err != nil {
		return ""
	}
	parsedBase, err := url.Parse(base)
	if err != nil {
		return ""
	}

	if parsedURL.IsAbs() {
		if parsedBase.Hostname() == parsedURL.Hostname() {
			return parsedURL.String()
		}
	} else {
		uri := parsedBase.ResolveReference(parsedURL)
		return uri.String()
	}

	return ""
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println(usage)
		os.Exit(1)
	}

	queue := make(chan string)
	filteredQueue := make(chan string)

	go func() { queue <- args[0] }()
	go filterQueue(queue, filteredQueue)

	for uri := range filteredQueue {
		if uri == "done" {
			fmt.Println("Fetching done!")
			os.Exit(0)
		}
		go enqueue(uri, queue)
	}
}
