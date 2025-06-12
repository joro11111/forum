package main

import (
	"fmt"
	"html/template"
	"literary-lions/database"
	"literary-lions/handlers"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Initialize database
	db, err := database.NewDB("forum.db")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize database tables
	if err := db.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Clean expired sessions periodically
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := db.CleanExpiredSessions(); err != nil {
					log.Printf("Error cleaning expired sessions: %v", err)
				}
			}
		}
	}()

	// Load templates
	templates, err := loadTemplates()
	if err != nil {
		log.Fatal("Failed to load templates:", err)
	}

	// Initialize handlers
	h := handlers.NewHandler(db, templates)

	// Setup routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", h.HomeHandler)
	mux.HandleFunc("/login", h.LoginHandler)
	mux.HandleFunc("/register", h.RegisterHandler)
	mux.HandleFunc("/logout", h.LogoutHandler)

	// Post routes
	mux.HandleFunc("/post/", h.ViewPostHandler)
	mux.HandleFunc("/create-post", h.CreatePostHandler)

	// Search routes
	mux.HandleFunc("/search", h.SearchHandler)
	mux.HandleFunc("/api/search-suggestions", h.SearchSuggestionsHandler)

	// Profile routes
	mux.HandleFunc("/profile/", h.ProfileHandler)
	mux.HandleFunc("/edit-profile", h.EditProfileHandler)
	mux.HandleFunc("/delete-profile", h.DeleteProfileHandler)

	// Admin routes (protected by admin middleware)
	mux.HandleFunc("/admin", h.AdminMiddleware(h.AdminPanelHandler))
	mux.HandleFunc("/admin/suspend", h.AdminMiddleware(h.AdminSuspendUserHandler))
	mux.HandleFunc("/admin/delete", h.AdminMiddleware(h.AdminDeleteUserHandler))

	// Comment and like routes (require authentication)
	mux.HandleFunc("/create-comment", h.CreateCommentHandler)
	mux.HandleFunc("/like-post", h.LikePostHandler)
	mux.HandleFunc("/like-comment", h.LikeCommentHandler)

	// Static files (CSS, JS, images) - if needed in the future
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// 404 handler
	mux.HandleFunc("/404", h.NotFoundHandler)

	// Wrap with logging middleware
	handler := loggingMiddleware(mux)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🦁 Literary Lions Forum starting on port %s", port)
	log.Printf("📖 Visit http://localhost:%s to start your literary journey!", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// loadTemplates loads and parses all HTML templates
func loadTemplates() (*template.Template, error) {
	// Create a new template with custom functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"slice": func(s string, start, end int) string {
			if start < 0 {
				start = 0
			}
			if end > len(s) {
				end = len(s)
			}
			if start >= end {
				return ""
			}
			return s[start:end]
		},
		"printf": func(format string, args ...interface{}) string {
			return fmt.Sprintf(format, args...)
		},
		"add": func(a, b int) int {
			return a + b
		},
	})

	// Collect all template files
	var templateFiles []string
	err := filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".html") {
			templateFiles = append(templateFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Parse all template files together
	if len(templateFiles) > 0 {
		tmpl, err = tmpl.ParseFiles(templateFiles...)
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v %s", r.Method, r.URL.Path, ww.statusCode, duration, r.RemoteAddr)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}
