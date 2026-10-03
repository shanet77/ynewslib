package hn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// BaseURL of the official read-only Firebase API. Var so tests can override.
var BaseURL = "https://hacker-news.firebaseio.com/v0"

type Item struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	By       string `json:"by"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Text     string `json:"text"`
	Score    int    `json:"score"`
	Time     int64  `json:"time"`
	Desc     int    `json:"descendants"`
	Kids     []int  `json:"kids"`
	Parent   int    `json:"parent"`
	Dead     bool   `json:"dead"`
	Children []Item `json:"-"`
}

type User struct {
	ID      string `json:"id"`
	Karma   int    `json:"karma"`
	Created int64  `json:"created"`
	About   string `json:"about"`
}

type Client struct {
	http  *http.Client
	items *cache[int, Item]
	users *cache[string, User]
}

func NewClient() *Client {
	return &Client{
		http:  &http.Client{Timeout: 10 * time.Second, Transport: limitedTransport(50)},
		items: newCache[int, Item](60 * time.Second),
		users: newCache[string, User](5 * time.Minute),
	}
}

// limitedTransport caps concurrent outbound requests globally, so many
// simultaneous visitors can't multiply the per-thread fan-out into an
// unbounded hammering of the HN API.
func limitedTransport(max int) http.RoundTripper {
	rt := http.DefaultTransport
	sem := make(chan struct{}, max)
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		sem <- struct{}{}
		defer func() { <-sem }()
		return rt.RoundTrip(r)
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func (c *Client) Item(id int) (Item, error) {
	if it, ok := c.items.get(id); ok {
		return it, nil
	}
	var it Item
	if err := c.getJSON(fmt.Sprintf("/item/%d.json", id), &it); err != nil {
		return it, err
	}
	c.items.set(id, it)
	return it, nil
}

func (c *Client) User(name string) (User, error) {
	if u, ok := c.users.get(name); ok {
		return u, nil
	}
	var u User
	if err := c.getJSON(fmt.Sprintf("/user/%s.json", url.PathEscape(name)), &u); err != nil {
		return u, err
	}
	c.users.set(name, u)
	return u, nil
}

// Stories returns one page (30) of a feed: top, new, best, ask, show, job.
func (c *Client) Stories(feed string, page int) ([]Item, error) {
	var ids []int
	if err := c.getJSON(fmt.Sprintf("/%sstories.json", feed), &ids); err != nil {
		return nil, err
	}
	start := page * 30
	if start >= len(ids) {
		return nil, nil
	}
	end := min(start+30, len(ids))
	ids = ids[start:end]

	items := make([]Item, len(ids))
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i, id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items[i], _ = c.Item(id)
		}(i, id)
	}
	wg.Wait()
	return items, nil
}

// Thread returns the story and its full comment tree, fetched concurrently.
func (c *Client) Thread(id int) (Item, []Item, error) {
	story, err := c.Item(id)
	if err != nil {
		return story, nil, err
	}
	sem := make(chan struct{}, 24)
	return story, c.tree(story.Kids, sem), nil
}

func (c *Client) tree(ids []int, sem chan struct{}) []Item {
	if len(ids) == 0 {
		return nil
	}
	out := make([]Item, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i, id int) {
			defer wg.Done()
			sem <- struct{}{}
			it, err := c.Item(id)
			<-sem
			if err != nil || it.Dead || it.Type == "" {
				return
			}
			it.Children = c.tree(it.Kids, sem)
			it.Kids = nil
			out[i] = it
		}(i, id)
	}
	wg.Wait()
	return out
}

func (c *Client) getJSON(path string, v any) error {
	resp, err := c.http.Get(BaseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hn api %s: %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
