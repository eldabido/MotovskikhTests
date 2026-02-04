package api

import (
	"net/http"
	"strings"
)

type wrapper func(http.Handler) http.Handler

func (m *Manager) wrapContentTypeJSON(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		// Разрешаем оба варианта
		if !strings.Contains(strings.ToLower(ct), "application/json") {
			m.sendErrorPage(w, http.StatusBadRequest)
			return
		}
		inner.ServeHTTP(w, r)
	})
}

func (m *Manager) wrapBodyMaxSize(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		inner.ServeHTTP(w, r)
	})
}

// wrapEasterEggHeader добавляет ржомбу в заголовки.
// nolint:canonicalheader
func (m *Manager) wrapEasterEggHeader(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("hey", "what are you trying to find here")
		w.Header().Set("Leon-Motovskikh", "is the best")
		w.Header().Set("x-files", "Scully approves")
		inner.ServeHTTP(w, r)
	})
}

// wrapRecover отправляет ошибку.
func wrapRecover(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer notifyRecover(map[string]any{"uri": r.RequestURI})
		h.ServeHTTP(w, r)
	})
}