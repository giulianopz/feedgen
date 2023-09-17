package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-faker/faker/v4/pkg/slice"
	"github.com/gorilla/feeds"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

type config struct {
	baseUrl string

	Title       string `yaml:"title,omitempty"`
	Link        string `yaml:"link,omitempty"`
	Description string `yaml:"description,omitempty"`
	AuthorName  string `yaml:"author_name,omitempty"`
	AuthorEmail string `yaml:"author_email,omitempty"`
	FeedFormat  string `yaml:"feed_format,omitempty"`

	ItemNode string   `yaml:"item_node,omitempty"`
	ItemKey  string   `yaml:"item_key,omitempty"`
	ItemVals []string `yaml:"item_vals,omitempty"`

	HrefNode string `yaml:"href_node,omitempty"`
}

var usage string = "USAGE: feedgen YAML_FILE [OUT_DIR]"

func main() {

	if len(os.Args) < 2 {
		fmt.Println(usage)
	}

	var (
		configFile = os.Args[1]
		outputDir  = "."
	)

	if len(os.Args) == 3 {
		outputDir = os.Args[2]
	}

	bs, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	config := new(config)
	if err = yaml.Unmarshal(bs, config); err != nil {
		panic(err)
	}

	url, err := url.Parse(config.Link)
	if err != nil {
		panic(err)
	}

	config.baseUrl = fmt.Sprintf("%s://%s", url.Scheme, url.Hostname())

	feed := &feeds.Feed{
		Title:       config.Title,
		Link:        &feeds.Link{Href: config.Link},
		Description: config.Description,
		Author:      &feeds.Author{Name: config.AuthorName, Email: config.AuthorEmail},
		Created:     time.Now(),
	}

	resp, err := http.Get(config.Link)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var lastDate time.Time = time.Now()

	var f func(*html.Node)
	f = func(n *html.Node) {
		// div or td, usally
		if n.Type == html.ElementNode && n.Data == config.ItemNode {

			if config.ItemKey == "" { // search in current node

				item := newFeedItem(n, config)
				if item.Link != nil && item.Link.Href != "" {
					item.Created = lastDate
					lastDate = lastDate.Add(-24 * time.Hour)
					feed.Items = append(feed.Items, item)
				}

			} else { // search in child node
				for _, a := range n.Attr {

					if a.Key == config.ItemKey && slice.Contains(config.ItemVals, a.Val) {

						item := newFeedItem(n, config)
						if item.Link != nil && item.Link.Href != "" {
							item.Created = lastDate
							lastDate = lastDate.Add(-24 * time.Hour)
							feed.Items = append(feed.Items, item)
						}
						break
					}
				}
			}

		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	var result string
	switch config.FeedFormat {
	case "rss":
		{
			rss, err := feed.ToRss()
			if err != nil {
				panic(err)
			}
			result = rss
		}
	case "atom":
		{
			atom, err := feed.ToAtom()
			if err != nil {
				panic(err)
			}
			result = atom
		}
	}

	if !strings.HasSuffix(outputDir, "/") {
		outputDir += "/"
	}

	fileName := regexp.MustCompile("[^a-zA-Z]").ReplaceAllString(strings.ToLower(feed.Title), "-")
	fileName = regexp.MustCompile(`\-+`).ReplaceAllString(fileName, "-")
	if err := os.WriteFile(outputDir+fileName+".xml", []byte(result), os.ModePerm); err != nil {
		panic(err)
	}
}

func newFeedItem(node *html.Node, config *config) *feeds.Item {

	item := &feeds.Item{
		Author: &feeds.Author{Name: config.AuthorName},
	}

	var finder func(*html.Node)
	finder = func(n *html.Node) {

		// a
		if n.Type == html.ElementNode && n.Data == config.HrefNode {
			for _, a := range n.Attr {
				if a.Key == "href" {

					link := a.Val
					if !strings.HasPrefix(link, "http") {
						if !strings.HasPrefix(link, "/") {
							link = "/" + link
						}
						link = config.baseUrl + link
					}

					item.Link = &feeds.Link{Href: link}
					item.Title = n.FirstChild.Data
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			finder(child)
		}
	}

	finder(node)

	return item
}
