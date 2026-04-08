package handlers

import (
	"context"
	"net/http"

	"bootcamp/web/internal/db"
	"bootcamp/web/internal/session"
)

type contextKey int

const userContextKey contextKey = 0

func userFromContext(r *http.Request) *db.User {
	u, _ := r.Context().Value(userContextKey).(*db.User)
	return u
}

func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sid := session.Get(r)
		if sid == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		user, err := a.DB.GetSessionUser(r.Context(), sid)
		if err != nil || user == nil {
			session.Clear(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := userFromContext(r)
		if user == nil || !user.IsAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
