package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseCustomTheme(t *testing.T) {
	ok := parseCustomTheme("bg=#14171a,fg=d6d3cd,dim=#8a8f98")
	if ok == nil || ok["bg"] != "#14171a" || ok["fg"] != "#d6d3cd" {
		t.Fatalf("valid custom theme rejected: %v", ok)
	}
	for _, bad := range []string{
		"bg=red",            // not hex
		"bg=#12345",         // bad length
		"evil=onload=alert", // unknown var name
		"bg=#fff,junk",      // malformed pair
	} {
		if parseCustomTheme(bad) != nil {
			t.Errorf("accepted invalid theme %q", bad)
		}
	}
}

func TestSetTheme(t *testing.T) {
	h := http.HandlerFunc(setTheme)

	r := httptest.NewRequest("GET", "/theme/light?back=/item/1", nil)
	r.SetPathValue("name", "light")
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/item/1" {
		t.Errorf("redirect: code=%d loc=%q", w.Code, w.Header().Get("Location"))
	}
	if c := w.Result().Cookies()[0]; c.Name != "theme" || c.Value != "light" {
		t.Errorf("cookie: %+v", c)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/theme/light?back=//evil.com", nil)
	r.SetPathValue("name", "light")
	h(w, r)
	if w.Header().Get("Location") != "/" {
		t.Errorf("protocol-relative back accepted: %q", w.Header().Get("Location"))
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/theme/custom?bg=1a1b26&link=7aa2f7&back=/", nil)
	r.SetPathValue("name", "custom")
	h(w, r)
	c := w.Result().Cookies()[0]
	if c.Value != "bg=1a1b26,link=7aa2f7" {
		t.Errorf("custom cookie: %q", c.Value)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/theme/nonsense", nil)
	r.SetPathValue("name", "nonsense")
	h(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown theme: code=%d", w.Code)
	}
}

func TestThemeCSS(t *testing.T) {
	get := func(cookie string) string {
		r := httptest.NewRequest("GET", "/theme.css", nil)
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "theme", Value: cookie})
		}
		w := httptest.NewRecorder()
		themeCSS(w, r)
		return w.Body.String()
	}
	if out := get(""); out != "" {
		t.Errorf("auto should emit nothing, got %q", out)
	}
	out := get("dark")
	if !strings.Contains(out, "--bg:#14171a") || !strings.Contains(out, "--accent:#ff6600") {
		t.Errorf("dark missing vars: %q", out)
	}
	if out := get("bg=#010203"); !strings.Contains(out, "--bg:#010203") || strings.Contains(out, "--fg") {
		t.Errorf("custom partial: %q", out)
	}
	if out := get("bg=javascript:alert(1)"); out != "" {
		t.Errorf("invalid cookie leaked into css: %q", out)
	}
}
