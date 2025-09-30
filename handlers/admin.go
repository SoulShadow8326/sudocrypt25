package handlers

import "strings"

type Admins struct {
	set map[string]struct{}
}

func NewAdmins(raw string) *Admins {
	a := &Admins{set: make(map[string]struct{})}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return a
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		raw = strings.TrimPrefix(strings.TrimSuffix(raw, "]"), "[")
	}
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		e := strings.ToLower(strings.TrimSpace(p))
		e = strings.Trim(e, "\"' ")
		if e != "" {
			a.set[e] = struct{}{}
		}
	}
	return a
}

func (a *Admins) IsAdmin(email string) bool {
	if a == nil || len(a.set) == 0 {
		return false
	}
	e := strings.ToLower(strings.TrimSpace(email))
	_, ok := a.set[e]
	return ok
}
