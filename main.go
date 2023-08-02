package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/feeds"
	"golang.org/x/net/html"
)

func main() {

	feed := &feeds.Feed{
		Title:       "The Bucklog: Essays and Rants",
		Link:        &feeds.Link{Href: "https://weblog.jamisbuck.org/essays-and-rants/"},
		Description: "Assorted ramblings by Jamis Buck",
		Author:      &feeds.Author{Name: "Jamis Buck", Email: "jamis@jamisbuck.org"},
		Created:     time.Now(),
	}

	resp, err := http.Get("https://weblog.jamisbuck.org/essays-and-rants/")
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
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" && (a.Val == "post glimpse" || a.Val == "post link") {

					item := findArticle(n)

					if !item.Created.IsZero() {
						lastDate = item.Created
					} else if !lastDate.IsZero() {
						item.Created = lastDate.Add(-24 * time.Hour)
					} else {
						lastDate = time.Date(1970, 1, 1, 1, 1, 1, 1, time.UTC)
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

	rss, err := feed.ToRss()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rss)
}

func findArticle(node *html.Node) *feeds.Item {

	item := &feeds.Item{
		Author: &feeds.Author{Name: "Jamis Buck"},
	}

	var month, year string

	var finder func(*html.Node)
	finder = func(n *html.Node) {

		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					//fmt.Println(a.Val)
					item.Link = &feeds.Link{Href: "https://weblog.jamisbuck.org" + a.Val}
					//fmt.Println(n.FirstChild.Data)
					item.Title = n.FirstChild.Data
				}
			}
		}

		if n.Type == html.ElementNode && n.Data == "span" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "month" {
					//fmt.Println(n.FirstChild.Data)
					month = n.FirstChild.Data
				}
				if a.Key == "class" && a.Val == "year" {
					//fmt.Println(n.FirstChild.Data)
					year = n.FirstChild.Data
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			finder(child)
		}
	}

	finder(node)

	date, err := time.Parse("02-Jan-2006", fmt.Sprintf("%s-%s-%s", "01", month, year))
	if err != nil {
		//fmt.Println(err)
	} else {
		item.Created = date
	}

	return item
}
