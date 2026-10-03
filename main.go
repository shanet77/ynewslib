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
	Flash    string
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
		"age": age, "domain": domain, "sanitize": sanitize,
	}).ParseFS(files, "templates/*.html"))

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(files))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { list(w, tpl, "top", r) })
	mux.HandleFunc("GET /{feed}", func(w http.ResponseWriter, r *http.Request) { list(w, tpl, r.PathValue("feed"), r) })
	mux.HandleFunc("GET /item/{id}", itemHandler(tpl))
	mux.HandleFunc("GET /user/{name}", userHandler(tpl))
	mux.HandleFunc("GET /login", loginForm(tpl))
	mux.HandleFunc("POST /login", loginPost)
	mux.HandleFunc("POST /vote/{id}", votePost)

	log.Printf("ynews listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func list(w http.ResponseWriter, tpl *template.Template, feed string, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("p"))
	if err != nil || page < 0 {
		page = 0
	}
	rows, err := client.Stories(feed, page)
	if err != nil {
		http.Error(w, "HN API error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if feed == "" {
		feed = "top"
	}
	p := Page{Title: feed, Rows: rows, Next: page + 1}
	p.setNav(feed)
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
			http.Error(w, "HN API error: "+err.Error(), http.StatusBadGateway)
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
			http.Error(w, "HN API error: "+err.Error(), http.StatusBadGateway)
			return
		}
		p := Page{Title: u.ID, User: u}
		p.setNav("")
		if err := tpl.ExecuteTemplate(w, "user.html", p); err != nil {
			log.Print(err)
		}
	}
}

func loginForm(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := Page{Title: "login"}
		p.setNav("")
		if err := tpl.ExecuteTemplate(w, "login.html", p); err != nil {
			log.Print(err)
		}
	}
}

func loginPost(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	_, err := hn.Login(r.FormValue("username"), r.FormValue("password"))
	http.Error(w, "login not implemented yet (M3): "+err.Error(), http.StatusNotImplemented)
}

func votePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<span class="voted" title="voting lands in M3">▲</span>`)
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
