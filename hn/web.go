package hn

import "errors"

// M3 stubs: real implementations proxy news.ycombinator.com with the
// user's session cookie. Login POSTs to HN /login; Vote extracts the
// per-item auth token from the item page then hits /vote?...&auth=...
var ErrNotImplemented = errors.New("not implemented yet (M3)")

type Session struct {
	Cookie string
}

func Login(user, pass string) (Session, error) {
	return Session{}, ErrNotImplemented
}

func (s *Session) Vote(id int) error {
	return ErrNotImplemented
}
