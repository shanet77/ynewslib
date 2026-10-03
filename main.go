package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"ynewslib/hn"
)

//go:embed templates static
var files embed.FS

var (
	client    = hn.NewClient()
	sanitizer = newSanitizer()
)

var feeds = []struct{ Path, Label string }{
	{"", "top"}, {"new", "new"}, {"best", "best"}, {"ask", "ask"}, {"show", "show"}, {"jobs", "jobs"},
}

// user-supplied feed strings must map through this allowlist to Firebase feed names
var feedAPI = map[string]string{
	"": "top", "top": "top", "new": "new", "best": "best", "ask": "ask", "show": "show", "jobs": "job",
}

type Feed struct{ Path, Label string }

type Page struct {
	Title    string
	Nav      []Feed
	Active   string
	Rows     []hn.Item
	Next     int
	Story    hn.Item
	Comments []hn.Item
	User     hn.User
}

func (p *Page) setNav(active string) {
	p.Nav = make([]Feed, len(feeds))
	for i, f := range feeds {
		p.Nav[i] = Feed{f.Path, f.Label}
	}
	p.Active = active
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	tpl := template.Must(template.New("").Funcs(template.FuncMap{
		"age": age, "domain": domain, "sanitize": sanitize, "add": func(a, b int) int { return a + b },
	}).ParseFS(files, "templates/*.html"))

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(files))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { list(w, tpl, "top", r) })
	mux.HandleFunc("GET /{feed}", func(w http.ResponseWriter, r *http.Request) { list(w, tpl, r.PathValue("feed"), r) })
	mux.HandleFunc("GET /item/{id}", itemHandler(tpl))
	mux.HandleFunc("GET /user/{name}", userHandler(tpl))

	log.Printf("ynews listening on :%s", port)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           secureHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func apiErr(w http.ResponseWriter, err error) {
	log.Print(err)
	http.Error(w, "HN API error, try again", http.StatusBadGateway)
}

func list(w http.ResponseWriter, tpl *template.Template, feed string, r *http.Request) {
	api, ok := feedAPI[feed]
	if !ok {
		http.NotFound(w, r)
		return
	}
	page, err := strconv.Atoi(r.URL.Query().Get("p"))
	if err != nil || page < 0 {
		page = 0
	}
	rows, err := client.Stories(api, page)
	if err != nil {
		apiErr(w, err)
		return
	}
	active := feed
	if r.URL.Path == "/" {
		active = "top"
	}
	p := Page{Title: active, Rows: rows, Next: page + 1}
	p.setNav(active)
	if err := tpl.ExecuteTemplate(w, "list.html", p); err != nil {
		log.Print(err)
	}
}

func itemHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		story, comments, err := client.Thread(id)
		if err != nil {
			apiErr(w, err)
			return
		}
		p := Page{Title: story.Title, Story: story, Comments: comments}
		p.setNav("")
		if err := tpl.ExecuteTemplate(w, "thread.html", p); err != nil {
			log.Print(err)
		}
	}
}

func userHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		u, err := client.User(name)
		if err != nil {
			apiErr(w, err)
			return
		}
		p := Page{Title: u.ID, User: u}
		p.setNav("")
		if err := tpl.ExecuteTemplate(w, "user.html", p); err != nil {
			log.Print(err)
		}
	}
}

func age(t int64) string {
	d := time.Since(time.Unix(t, 0))
	switch {
	case d.Hours() >= 24:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d.Hours() >= 1:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
}

func domain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Host, "www.")
}

func sanitize(s string) template.HTML {
	return template.HTML(sanitizer.Sanitize(s))
}

func newSanitizer() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "i", "em", "b", "strong", "code", "pre", "blockquote", "ul", "ol", "li")
	p.AllowAttrs("href").OnElements("a")
	p.AllowStandardURLs()
	return p
}
