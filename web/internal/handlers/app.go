package handlers

import (
	"io/fs"
	"log/slog"
	"net/http"

	"bootcamp/web/internal/config"
	"bootcamp/web/internal/db"
	"bootcamp/web/internal/upload"
)

// App holds application dependencies and registers HTTP routes.
type App struct {
	DB     *db.DB
	Upload *upload.Client
	Cfg    *config.Config
	Files  fs.FS
	Log    *slog.Logger
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	// Static assets (CSS, JS)
	sub, _ := fs.Sub(a.Files, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static", http.FileServer(http.FS(sub))))

	// Public routes
	mux.HandleFunc("GET /login", a.handleLoginPage)
	mux.HandleFunc("POST /login", a.handleLogin)
	mux.HandleFunc("POST /logout", a.handleLogout)
	mux.HandleFunc("GET /healthz", a.handleHealth)

	// Authenticated routes
	mux.Handle("GET /{$}", a.requireAuth(http.HandlerFunc(a.handleIndex)))
	mux.Handle("POST /upload", a.requireAuth(http.HandlerFunc(a.handleUpload)))
	mux.Handle("GET /api/files", a.requireAuth(http.HandlerFunc(a.handleListFiles)))
	mux.Handle("GET /api/me", a.requireAuth(http.HandlerFunc(a.handleMe)))
	mux.Handle("GET /files/{name}", a.requireAuth(http.HandlerFunc(a.handleFileProxy)))
	mux.Handle("DELETE /files/{name}", a.requireAuth(http.HandlerFunc(a.handleFileDelete)))
	mux.Handle("POST /api/password", a.requireAuth(http.HandlerFunc(a.handleChangePassword)))

	// Admin routes
	admin := func(h http.Handler) http.Handler { return a.requireAuth(a.requireAdmin(h)) }
	mux.Handle("GET /admin", admin(http.HandlerFunc(a.handleAdminPage)))
	mux.Handle("GET /api/admin/users", admin(http.HandlerFunc(a.handleListUsers)))
	mux.Handle("POST /api/admin/users", admin(http.HandlerFunc(a.handleCreateUser)))
	mux.Handle("DELETE /api/admin/users/{id}", admin(http.HandlerFunc(a.handleDeleteUser)))
}
