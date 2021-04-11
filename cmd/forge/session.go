package main

import (
	"net/http"
)

func setSession(w http.ResponseWriter, session map[string]string) error {
	encoded, err := secureCookie.Encode("session", session)
	if err != nil {
		return err
	}
	c := &http.Cookie{
		Name:  "session",
		Value: encoded,
		Path:  "/",
	}
	http.SetCookie(w, c)
	return nil
}

func getSession(r *http.Request) (map[string]string, error) {
	value := make(map[string]string)
	c, _ := r.Cookie("session")
	if c == nil {
		return value, nil
	}
	err := secureCookie.Decode("session", c.Value, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func clearSession(w http.ResponseWriter) {
	c := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, c)
}
