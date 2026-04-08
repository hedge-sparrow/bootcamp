package handlers

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"time"

	"bootcamp/web/internal/session"
	"golang.org/x/crypto/bcrypt"
)

func (a *App) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	serveTemplate(w, a.Files, "templates/login.html")
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := a.DB.GetUserByUsername(r.Context(), username)
	if err != nil || user == nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	expires := time.Now().Add(a.Cfg.SessionDuration)
	sid, err := a.DB.CreateSession(r.Context(), user.ID, expires)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	session.Set(w, sid, expires, a.Cfg.CookieSecure)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	sid := session.Get(r)
	if sid != "" {
		a.DB.DeleteSession(r.Context(), sid)
	}
	session.Clear(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Current == "" || req.New == "" {
		http.Error(w, "current and new password are required", http.StatusBadRequest)
		return
	}

	user := userFromContext(r)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Current)); err != nil {
		http.Error(w, "current password is incorrect", http.StatusUnauthorized)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := a.DB.UpdatePassword(r.Context(), user.ID, string(hash)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serveTemplate(w http.ResponseWriter, files fs.FS, name string) {
	data, err := fs.ReadFile(files, name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
