package hn

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestThreadTree(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/item/", func(w http.ResponseWriter, r *http.Request) {
		var id int
		fmt.Sscanf(r.URL.Path, "/item/%d.json", &id)
		items := map[int]string{
			1:  `{"id":1,"type":"story","title":"t","kids":[10,20],"descendants":3}`,
			10: `{"id":10,"type":"comment","parent":1,"kids":[11]}`,
			11: `{"id":11,"type":"comment","parent":10}`,
			20: `{"id":20,"type":"comment","parent":1}`,
		}
		if s, ok := items[id]; ok {
			fmt.Fprint(w, s)
		} else {
			http.Error(w, "nope", 404)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	BaseURL = srv.URL

	story, comments, err := NewClient().Thread(1)
	if err != nil {
		t.Fatal(err)
	}
	if story.ID != 1 || len(comments) != 2 {
		t.Fatalf("story=%+v comments=%d", story, len(comments))
	}
	if len(comments[0].Children) != 1 || comments[0].Children[0].ID != 11 {
		t.Fatalf("nested comment missing: %+v", comments[0])
	}
	if comments[1].ID != 20 || len(comments[1].Children) != 0 {
		t.Fatalf("second top-level comment wrong: %+v", comments[1])
	}
}
