package main

import (
	"crypto/hmac"
	"net/http"
	"strings"
)

const cookieName = "session"

func authMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil || !hmac.Equal([]byte(cookie.Value), []byte(secret)) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusFound)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleLoginPage(loginHTML []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(loginHTML)
	}
}

func makeLoginHandler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !hmac.Equal([]byte(r.FormValue("secret")), []byte(secret)) {
			http.Redirect(w, r, "/login?err=1", http.StatusFound)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    secret,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   60 * 60 * 24 * 365,
		})
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
