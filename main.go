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

	NoDate       bool   `yaml:"no_date,omitempty"`
	SplittedDate bool   `yaml:"splitted_date,omitempty"`
	DateFormat   string `yaml:"date_format,omitempty"`
	FmtFormat    string `yaml:"fmt_format,omitempty"`

	DateNode string `yaml:"date_node,omitempty"`
	DateKey  string `yaml:"date_key,omitempty"`
	DateVal  string `yaml:"date_val,omitempty"`

	DayNode    string `yaml:"day_node,omitempty"`
	DayKey     string `yaml:"day_key,omitempty"`
	DayVal     string `yaml:"day_val,omitempty"`
	DayDefault string `yaml:"day_default,omitempty"`

	MonthNode    string `yaml:"month_node,omitempty"`
	MonthKey     string `yaml:"month_key,omitempty"`
	MonthVal     string `yaml:"month_val,omitempty"`
	MonthDefault string `yaml:"month_default,omitempty"`

	YearNode    string `yaml:"year_node,omitempty"`
	YearKey     string `yaml:"year_key,omitempty"`
	YearVal     string `yaml:"year_val,omitempty"`
	YearDefault string `yaml:"year_default,omitempty"`
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

	var lastDate time.Time

	var f func(*html.Node)
	f = func(n *html.Node) {
		// div
		if n.Type == html.ElementNode && n.Data == config.ItemNode {
			for _, a := range n.Attr {
				// class
				// "post glimpse", "post link"
				if a.Key == config.ItemKey && slice.Contains(config.ItemVals, a.Val) {

					item := NewFeedItem(n, config)

					if !item.Created.IsZero() {
						lastDate = item.Created
					} else if !lastDate.IsZero() {
						item.Created = lastDate.Add(-24 * time.Hour)
					} else {
						parsed, err := time.Parse(config.DateFormat, fmt.Sprintf(config.FmtFormat, config.DayDefault, config.MonthDefault, config.YearDefault))
						if err != nil {
							panic(err)
						}
						lastDate = parsed
					}

					feed.Items = append(feed.Items, item)
					break
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

func NewFeedItem(node *html.Node, config *config) *feeds.Item {

	item := &feeds.Item{
		Author: &feeds.Author{Name: config.AuthorName},
	}

	var day, month, year string
	var date time.Time = time.Now()

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

		if !config.NoDate && config.SplittedDate {
			if n.Type == html.ElementNode {

				if n.Data == config.YearNode {
					for _, a := range n.Attr {
						if a.Key == config.YearKey && a.Val == config.YearVal {
							year = n.FirstChild.Data
						}
					}
				}
				if n.Data == config.MonthNode {
					for _, a := range n.Attr {
						if a.Key == config.MonthKey && a.Val == config.MonthVal {
							month = n.FirstChild.Data
						}
					}
				}
				if n.Data == config.DayNode {
					for _, a := range n.Attr {
						if a.Key == config.DayKey && a.Val == config.DayVal {
							day = n.FirstChild.Data
						}
					}
				}

			}
		} else if !config.NoDate {
			if n.Type == html.ElementNode && n.Data == config.DateNode {
				for _, a := range n.Attr {
					if a.Key == config.DateKey && a.Val == config.DateVal {
						parsed, err := time.Parse(config.DateFormat, n.FirstChild.Data)
						if err != nil {
							panic(err)
						}
						date = parsed
					}
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			finder(child)
		}
	}

	finder(node)

	if !config.NoDate && config.SplittedDate {

		d, m, y := day, month, year
		if d == "" {
			d = config.DayDefault
		}
		if m == "" {
			m = config.MonthDefault
		}
		if y == "" {
			y = config.YearDefault
		}

		parsed, err := time.Parse(config.DateFormat, fmt.Sprintf(config.FmtFormat, d, m, y))
		if err != nil {
			panic(err)
		}
		date = parsed
	}

	item.Created = date

	return item
}
