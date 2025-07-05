package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"literary-lions/auth"
	"literary-lions/database"
	"literary-lions/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// PageData represents the common data structure for all templates
type PageData struct {
	Posts         []models.Post        `json:"posts,omitempty"`
	Categories    []models.Category    `json:"categories,omitempty"`
	Post          *models.Post         `json:"post,omitempty"`
	Comments      []models.Comment     `json:"comments,omitempty"`
	CommentTrees  []models.CommentTree `json:"comment_trees,omitempty"`
	CurrentUser   *models.User         `json:"current_user,omitempty"`
	Filter        string               `json:"filter,omitempty"`
	CategoryID    string               `json:"category_id,omitempty"`
	Title         string               `json:"title,omitempty"`
	Error         string               `json:"error,omitempty"`
	FormData      map[string]string    `json:"form_data,omitempty"`
	TotalComments int                  `json:"total_comments,omitempty"`
}

type Handler struct {
	DB        *database.DB
	Templates *template.Template
}

// NewHandler creates a new handler instance
func NewHandler(db *database.DB, templates *template.Template) *Handler {
	return &Handler{
		DB:        db,
		Templates: templates,
	}
}

// Middleware for authentication
func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.GetCurrentUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// GetCurrentUser retrieves the current user from session
func (h *Handler) GetCurrentUser(r *http.Request) *models.User {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	session, err := h.DB.GetSessionByUUID(cookie.Value)
	if err != nil {
		return nil
	}

	user, err := h.DB.GetUserByID(session.UserID)
	if err != nil {
		return nil
	}

	return user
}

func (h *Handler) countTotalComments(commentTrees []models.CommentTree) int {
	total := 0
	for _, tree := range commentTrees {
		total += 1 + h.countCommentsInTree(tree)
	}
	return total
}

func (h *Handler) countCommentsInTree(tree models.CommentTree) int {
	count := 0
	for _, reply := range tree.Replies {
		count += 1 + h.countCommentsInTree(reply)
	}
	return count
}

func (h *Handler) buildCommentTree(comments []models.Comment) []models.CommentTree {
	// Create a map to store comments by their ID for quick lookup
	commentMap := make(map[int]models.Comment)
	var topLevelComments []models.Comment

	// First pass: create comment map and identify top-level comments
	for _, comment := range comments {
		commentMap[comment.ID] = comment
		if comment.ParentID == nil {
			topLevelComments = append(topLevelComments, comment)
		}
	}

	// Build the tree recursively
	var result []models.CommentTree
	for _, comment := range topLevelComments {
		tree := h.buildCommentSubtree(comment, commentMap)
		result = append(result, tree)
	}

	return result
}

// Helper function to recursively build comment subtree
func (h *Handler) buildCommentSubtree(comment models.Comment, commentMap map[int]models.Comment) models.CommentTree {
	var replies []models.CommentTree

	// Find all direct replies to this comment
	for _, c := range commentMap {
		if c.ParentID != nil && *c.ParentID == comment.ID {
			// Recursively build subtree for this reply
			subtree := h.buildCommentSubtree(c, commentMap)
			replies = append(replies, subtree)
		}
	}

	return models.CommentTree{
		Comment: comment,
		Replies: replies,
	}
}

// LoadPageTemplate loads the base template and a specific page template
func (h *Handler) LoadPageTemplate(templateFile string) (*template.Template, error) {
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
		"countComments": func(commentTrees []models.CommentTree) int {
			count := 0
			for _, tree := range commentTrees {
				count += 1 + h.countCommentsInTree(tree)
			}
			return count
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				panic("dict requires an even number of arguments")
			}
			result := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					panic("dict keys must be strings")
				}
				result[key] = values[i+1]
			}
			return result
		},
	})

	// Parse base template and the specific page template
	tmpl, err := tmpl.ParseFiles("templates/base.html", templateFile)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

// Home page handler
func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.NotFoundHandler(w, r)
		return
	}

	var posts []models.Post
	var err error
	var categories []models.Category
	currentUser := h.GetCurrentUser(r)

	// Get categories for filter
	categories, err = h.DB.GetAllCategories()
	if err != nil {
		http.Error(w, "Error fetching categories", http.StatusInternalServerError)
		return
	}

	// Handle filtering
	filter := r.URL.Query().Get("filter")
	categoryID := r.URL.Query().Get("category")

	// Check if current user is admin to decide whether to show suspended content
	showSuspended := currentUser != nil && currentUser.IsAdmin()

	switch filter {
	case "my-posts":
		if currentUser != nil {
			posts, err = h.DB.GetPostsByUser(currentUser.ID)
		}
	case "liked-posts":
		if currentUser != nil {
			posts, err = h.DB.GetLikedPostsByUser(currentUser.ID)
		}
	default:
		if categoryID != "" {
			catID, parseErr := strconv.Atoi(categoryID)
			if parseErr == nil {
				posts, err = h.DB.GetPostsByCategory(catID)
			} else {
				posts, err = h.DB.GetPostsWithSuspendedFilter(showSuspended)
			}
		} else {
			posts, err = h.DB.GetPostsWithSuspendedFilter(showSuspended)
		}
	}

	if err != nil {
		http.Error(w, "Error fetching posts", http.StatusInternalServerError)
		return
	}

	// Check if user was just deleted
	var successMessage string
	if r.URL.Query().Get("deleted") == "true" {
		successMessage = "Profile successfully deleted. Thank you for being part of Literary Lions!"
	}

	data := PageData{
		Posts:       posts,
		Categories:  categories,
		CurrentUser: currentUser,
		Filter:      filter,
		CategoryID:  categoryID,
		Title:       "Home",
		FormData: map[string]string{
			"success": successMessage,
		},
	}

	tmpl, err := h.LoadPageTemplate("templates/index.html")
	if err != nil {
		log.Printf("Failed to load index template: %v", err)
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

// Login handlers
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		data := PageData{
			Title: "Login",
		}

		tmpl, err := h.LoadPageTemplate("templates/login.html")
		if err != nil {
			log.Printf("Failed to load login template: %v", err)
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			log.Printf("Login template execution error: %v", err)
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")

		if email == "" || password == "" {
			data := PageData{
				Error: "Email and password are required",
				Title: "Login",
			}

			tmpl, err := h.LoadPageTemplate("templates/login.html")
			if err != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		user, err := h.DB.GetUserByEmail(email)
		if err != nil || !auth.CheckPassword(password, user.Password) {
			data := PageData{
				Error: "Invalid email or password",
				Title: "Login",
			}

			tmpl, err := h.LoadPageTemplate("templates/login.html")
			if err != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		// Create session
		uuid, err := auth.GenerateUUID()
		if err != nil {
			http.Error(w, "Error creating session", http.StatusInternalServerError)
			return
		}

		session := &models.Session{
			UserID:    user.ID,
			UUID:      uuid,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		if err := h.DB.CreateSession(session); err != nil {
			http.Error(w, "Error creating session", http.StatusInternalServerError)
			return
		}

		// Set cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    uuid,
			Expires:  session.ExpiresAt,
			HttpOnly: true,
			Path:     "/",
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Register handlers
func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := PageData{
			Title: "Register",
		}

		tmpl, err := h.LoadPageTemplate("templates/register.html")
		if err != nil {
			log.Printf("Failed to load register template: %v", err)
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		email := strings.TrimSpace(r.FormValue("email"))
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		// Validation
		var errors []string

		if email == "" {
			errors = append(errors, "Email is required")
		} else if !auth.ValidateEmail(email) {
			errors = append(errors, "Invalid email format")
		}

		if username == "" {
			errors = append(errors, "Username is required")
		} else if err := auth.ValidateUsername(username); err != nil {
			errors = append(errors, err.Error())
		}

		if password == "" {
			errors = append(errors, "Password is required")
		} else if err := auth.ValidatePassword(password); err != nil {
			errors = append(errors, err.Error())
		}

		// Check for existing users
		emailExists, usernameExists, err := h.DB.CheckUserExists(email, username)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if emailExists {
			errors = append(errors, "Email already exists")
		}
		if usernameExists {
			errors = append(errors, "Username already exists")
		}

		if len(errors) > 0 {
			data := PageData{
				Error: strings.Join(errors, "; "),
				Title: "Register",
			}

			tmpl, err := h.LoadPageTemplate("templates/register.html")
			if err != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		// Hash password
		hashedPassword, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "Error processing password", http.StatusInternalServerError)
			return
		}

		// Create user
		user := &models.User{
			Username: username,
			Email:    email,
			Password: hashedPassword,
		}

		if err := h.DB.CreateUser(user); err != nil {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Logout handler
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.DB.DeleteSession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Create post handlers
func (h *Handler) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := h.GetCurrentUser(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		categories, err := h.DB.GetAllCategories()
		if err != nil {
			http.Error(w, "Error fetching categories", http.StatusInternalServerError)
			return
		}

		data := PageData{
			Categories:  categories,
			CurrentUser: currentUser,
			Title:       "Create Post",
		}

		tmpl, err := h.LoadPageTemplate("templates/create_post.html")
		if err != nil {
			log.Printf("Failed to load create_post template: %v", err)
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		title := strings.TrimSpace(r.FormValue("title"))
		content := strings.TrimSpace(r.FormValue("content"))
		categoryIDStr := r.FormValue("category_id")

		var errors []string

		if title == "" {
			errors = append(errors, "Title is required")
		}
		if content == "" {
			errors = append(errors, "Content is required")
		}

		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil || categoryID <= 0 {
			errors = append(errors, "Valid category is required")
		}

		if len(errors) > 0 {
			categories, _ := h.DB.GetAllCategories()
			data := PageData{
				Categories:  categories,
				CurrentUser: currentUser,
				Error:       strings.Join(errors, "; "),
				Title:       "Create Post",
			}
			tmpl, err := h.LoadPageTemplate("templates/create_post.html")
			if err != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		post := &models.Post{
			Title:      title,
			Content:    content,
			UserID:     currentUser.ID,
			CategoryID: categoryID,
		}

		if err := h.DB.CreatePost(post); err != nil {
			http.Error(w, "Error creating post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/post/%d", post.ID), http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// View post handler
func (h *Handler) ViewPostHandler(w http.ResponseWriter, r *http.Request) {
	postIDStr := strings.TrimPrefix(r.URL.Path, "/post/")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		h.NotFoundHandler(w, r)
		return
	}

	post, err := h.DB.GetPostByID(postID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.NotFoundHandler(w, r)
			return
		}
		http.Error(w, "Error fetching post", http.StatusInternalServerError)
		return
	}

	currentUser := h.GetCurrentUser(r)

	// Get comments for the post (filter suspended users unless admin)
	showSuspended := currentUser != nil && currentUser.IsAdmin()
	allComments, err := h.DB.GetCommentsWithSuspendedFilter(postID, showSuspended)
	if err != nil {
		http.Error(w, "Error fetching comments", http.StatusInternalServerError)
		return
	}

	// Build hierarchical comment tree
	commentTrees := h.buildCommentTree(allComments)

	data := PageData{
		Post:         post,
		Comments:     allComments,
		CommentTrees: commentTrees,
		CurrentUser:  currentUser,
		Title:        post.Title,
	}

	// Add total comments count to FormData for template access
	if data.FormData == nil {
		data.FormData = make(map[string]string)
	}
	data.FormData["total_comments"] = strconv.Itoa(len(allComments))

	tmpl, err := h.LoadPageTemplate("templates/post.html")
	if err != nil {
		log.Printf("Failed to load post template: %v", err)
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("Template execution error in ViewPostHandler: %v", err)
		log.Printf("Post ID: %d, CommentTrees count: %d", postID, len(commentTrees))
		// Don't try to send error response as headers may already be written
		return
	}
}

// Create comment handler
func (h *Handler) CreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser := h.GetCurrentUser(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postIDStr := r.FormValue("post_id")
	parentIDStr := r.FormValue("parent_id")
	content := strings.TrimSpace(r.FormValue("content"))

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	if content == "" {
		http.Error(w, "Comment content is required", http.StatusBadRequest)
		return
	}

	comment := &models.Comment{
		Content: content,
		UserID:  currentUser.ID,
		PostID:  postID,
	}

	// Handle parent ID for replies
	if parentIDStr != "" {
		parentID, err := strconv.Atoi(parentIDStr)
		if err != nil {
			http.Error(w, "Invalid parent ID", http.StatusBadRequest)
			return
		}
		comment.ParentID = &parentID
	}

	if err := h.DB.CreateComment(comment); err != nil {
		http.Error(w, "Error creating comment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

// Like post handler
func (h *Handler) LikePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser := h.GetCurrentUser(r)
	if currentUser == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	postIDStr := r.FormValue("post_id")
	action := r.FormValue("action")

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	isLike := action == "like"

	if err := h.DB.LikePost(currentUser.ID, postID, isLike); err != nil {
		http.Error(w, "Error processing like", http.StatusInternalServerError)
		return
	}

	// Redirect back to the post or referring page
	referer := r.Header.Get("Referer")
	if referer != "" {
		http.Redirect(w, r, referer, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
	}
}

// Like comment handler
func (h *Handler) LikeCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser := h.GetCurrentUser(r)
	if currentUser == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	commentIDStr := r.FormValue("comment_id")
	action := r.FormValue("action")

	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	isLike := action == "like"

	if err := h.DB.LikeComment(currentUser.ID, commentID, isLike); err != nil {
		http.Error(w, "Error processing like", http.StatusInternalServerError)
		return
	}

	// Redirect back to the referring page
	referer := r.Header.Get("Referer")
	if referer != "" {
		http.Redirect(w, r, referer, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// 404 handler
func (h *Handler) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	data := PageData{
		CurrentUser: h.GetCurrentUser(r),
		Title:       "Page Not Found",
	}

	tmpl, err := h.LoadPageTemplate("templates/404.html")
	if err != nil {
		log.Printf("Failed to load 404 template: %v", err)
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
	}
}

// Search handler
func (h *Handler) SearchHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := strings.TrimSpace(r.URL.Query().Get("q"))
	currentUser := h.GetCurrentUser(r)

	var posts []models.Post
	var err error

	if searchTerm != "" {
		posts, err = h.DB.SearchPosts(searchTerm, 50)
		if err != nil {
			http.Error(w, "Error searching posts", http.StatusInternalServerError)
			return
		}
	}

	categories, err := h.DB.GetAllCategories()
	if err != nil {
		http.Error(w, "Error fetching categories", http.StatusInternalServerError)
		return
	}

	data := PageData{
		Posts:       posts,
		Categories:  categories,
		CurrentUser: currentUser,
		Title:       "Search Results",
		Filter:      "search",
		FormData: map[string]string{
			"q": searchTerm,
		},
	}

	tmpl, err := h.LoadPageTemplate("templates/search.html")
	if err != nil {
		log.Printf("Failed to load search template: %v", err)
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Please enter search criteria", http.StatusInternalServerError)
	}
}

// Search suggestions API for real-time search
func (h *Handler) SearchSuggestionsHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := strings.TrimSpace(r.URL.Query().Get("q"))

	if searchTerm == "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	posts, err := h.DB.SearchPostSuggestions(searchTerm, 5)
	if err != nil {
		http.Error(w, "Error searching posts", http.StatusInternalServerError)
		return
	}

	// Create a simple response structure
	type suggestion struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}

	suggestions := make([]suggestion, 0, len(posts))
	for _, post := range posts {
		suggestions = append(suggestions, suggestion{
			ID:    post.ID,
			Title: post.Title,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	// Simple JSON encoding without external library
	response := "["
	for i, s := range suggestions {
		if i > 0 {
			response += ","
		}
		response += fmt.Sprintf(`{"id":%d,"title":"%s"}`, s.ID, strings.ReplaceAll(s.Title, `"`, `\"`))
	}
	response += "]"

	w.Write([]byte(response))
}

// Profile handler
func (h *Handler) ProfileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract username from URL path
	username := strings.TrimPrefix(r.URL.Path, "/profile/")

	// Get user by username
	user, err := h.DB.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			h.NotFoundHandler(w, r)
			return
		}
		http.Error(w, "Error fetching user", http.StatusInternalServerError)
		return
	}

	// Get user's posts
	posts, err := h.DB.GetPostsByUser(user.ID)
	if err != nil {
		http.Error(w, "Error fetching user posts", http.StatusInternalServerError)
		return
	}

	currentUser := h.GetCurrentUser(r)

	data := PageData{
		Posts:       posts,
		CurrentUser: currentUser,
		Title:       fmt.Sprintf("%s's Profile", user.Username),
	}

	// Add the profile user to the data structure
	type ProfilePageData struct {
		PageData
		ProfileUser *models.User `json:"profile_user"`
	}

	profileData := ProfilePageData{
		PageData:    data,
		ProfileUser: user,
	}

	tmpl, err := h.LoadPageTemplate("templates/profile.html")
	if err != nil {
		log.Printf("Failed to load profile template: %v", err)
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", profileData); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

// Edit profile handler
func (h *Handler) EditProfileHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := h.GetCurrentUser(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		data := PageData{
			CurrentUser: currentUser,
			Title:       "Edit Profile",
		}

		tmpl, err := h.LoadPageTemplate("templates/edit_profile.html")
		if err != nil {
			log.Printf("Failed to load edit profile template: %v", err)
			http.Error(w, "Error loading template", http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		profilePicture := strings.TrimSpace(r.FormValue("profile_picture"))
		signature := strings.TrimSpace(r.FormValue("signature"))

		// Basic validation for profile picture URL
		if profilePicture != "" && !strings.HasPrefix(profilePicture, "http") {
			data := PageData{
				CurrentUser: currentUser,
				Title:       "Edit Profile",
				Error:       "Profile picture must be a valid URL starting with http",
			}

			tmpl, err := h.LoadPageTemplate("templates/edit_profile.html")
			if err != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		if len(signature) > 500 {
			data := PageData{
				CurrentUser: currentUser,
				Title:       "Edit Profile",
				Error:       "Signature must be less than 500 characters",
			}

			tmpl, err := h.LoadPageTemplate("templates/edit_profile.html")
			if err != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		err := h.DB.UpdateUserProfile(currentUser.ID, profilePicture, signature)
		if err != nil {
			http.Error(w, "Error updating profile", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/profile/%s", currentUser.Username), http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Delete profile handler
func (h *Handler) DeleteProfileHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := h.GetCurrentUser(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		// Get confirmation from form
		confirmation := strings.TrimSpace(r.FormValue("confirmation"))

		// Check if user typed their username correctly for confirmation
		if confirmation != currentUser.Username {
			data := PageData{
				CurrentUser: currentUser,
				Title:       "Edit Profile",
				Error:       "Please type your username exactly to confirm deletion",
			}

			tmpl, err := h.LoadPageTemplate("templates/edit_profile.html")
			if err != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		// Delete the user and all related data
		err := h.DB.DeleteUser(currentUser.ID)
		if err != nil {
			log.Printf("Error deleting user %d: %v", currentUser.ID, err)
			data := PageData{
				CurrentUser: currentUser,
				Title:       "Edit Profile",
				Error:       "Failed to delete profile. Please try again.",
			}

			tmpl, err2 := h.LoadPageTemplate("templates/edit_profile.html")
			if err2 != nil {
				http.Error(w, "Error loading template", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			tmpl.ExecuteTemplate(w, "base", data)
			return
		}

		// Clear the session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		// Redirect to home page with success message
		http.Redirect(w, r, "/?deleted=true", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Admin middleware
func (h *Handler) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.GetCurrentUser(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if !user.IsAdmin() {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// Admin panel handler
func (h *Handler) AdminPanelHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := h.GetCurrentUser(r)
	if currentUser == nil || !currentUser.IsAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get all users
	users, err := h.DB.GetAllUsers()
	if err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}

	// Get user statistics for each user
	type UserWithStats struct {
		models.User
		PostsCount    int `json:"posts_count"`
		CommentsCount int `json:"comments_count"`
		LikesReceived int `json:"likes_received"`
	}

	var usersWithStats []UserWithStats
	for _, user := range users {
		posts, comments, likes, err := h.DB.GetUserStats(user.ID)
		if err != nil {
			log.Printf("Error getting stats for user %d: %v", user.ID, err)
			posts, comments, likes = 0, 0, 0
		}

		usersWithStats = append(usersWithStats, UserWithStats{
			User:          user,
			PostsCount:    posts,
			CommentsCount: comments,
			LikesReceived: likes,
		})
	}

	// Handle URL parameters for success/error messages
	var formData map[string]string
	if success := r.URL.Query().Get("success"); success != "" {
		formData = map[string]string{"success": success}
	} else if errorMsg := r.URL.Query().Get("error"); errorMsg != "" {
		formData = map[string]string{"error": errorMsg}
	}

	data := struct {
		PageData
		Users []UserWithStats `json:"users"`
	}{
		PageData: PageData{
			CurrentUser: currentUser,
			Title:       "Admin Panel",
			FormData:    formData,
		},
		Users: usersWithStats,
	}

	tmpl, err := h.LoadPageTemplate("templates/admin_panel.html")
	if err != nil {
		log.Printf("Failed to load admin panel template: %v", err)
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

// Admin suspend user handler
func (h *Handler) AdminSuspendUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser := h.GetCurrentUser(r)
	if currentUser == nil || !currentUser.IsAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")

	switch action {
	case "suspend":
		err = h.DB.SuspendUser(userID)
	case "unsuspend":
		err = h.DB.UnsuspendUser(userID)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error %s user %d: %v", action, userID, err)
		http.Error(w, fmt.Sprintf("Error %s user", action), http.StatusInternalServerError)
		return
	}

	// Redirect back to admin panel
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// Admin delete user handler
func (h *Handler) AdminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	currentUser := h.GetCurrentUser(r)
	if currentUser == nil || !currentUser.IsAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Prevent admin from deleting themselves or other admins
	targetUser, err := h.DB.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if targetUser.IsAdmin() {
		http.Error(w, "Cannot delete admin users", http.StatusForbidden)
		return
	}

	if targetUser.ID == currentUser.ID {
		http.Error(w, "Cannot delete yourself", http.StatusForbidden)
		return
	}

	// Confirmation check
	confirmation := r.FormValue("confirmation")
	if confirmation != targetUser.Username {
		http.Redirect(w, r, "/admin?error=confirmation", http.StatusSeeOther)
		return
	}

	// Delete the user and all related data
	err = h.DB.DeleteUser(userID)
	if err != nil {
		log.Printf("Error deleting user %d: %v", userID, err)
		http.Redirect(w, r, "/admin?error=delete", http.StatusSeeOther)
		return
	}

	// Redirect back to admin panel with success message
	http.Redirect(w, r, "/admin?success=deleted", http.StatusSeeOther)
}
