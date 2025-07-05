package database

import (
	"database/sql"
	"fmt"
	"literary-lions/auth"
	"literary-lions/models"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

// NewDB creates a new database connection
func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// InitDB initializes the database with required tables
func (db *DB) InitDB() error {
	// CREATE queries for all tables
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			profile_picture TEXT DEFAULT '',
			signature TEXT DEFAULT '',
			role TEXT DEFAULT 'user',
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id),
			FOREIGN KEY(category_id) REFERENCES categories(id)
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			parent_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id),
			FOREIGN KEY(post_id) REFERENCES posts(id),
			FOREIGN KEY(parent_id) REFERENCES comments(id)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			uuid TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS post_likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			is_like BOOLEAN NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id),
			FOREIGN KEY(post_id) REFERENCES posts(id),
			UNIQUE(user_id, post_id)
		)`,
		`CREATE TABLE IF NOT EXISTS comment_likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			comment_id INTEGER NOT NULL,
			is_like BOOLEAN NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id),
			FOREIGN KEY(comment_id) REFERENCES comments(id),
			UNIQUE(user_id, comment_id)
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("error creating table: %v", err)
		}
	}

	// Add migration for existing databases
	if err := db.migrateUserTable(); err != nil {
		return fmt.Errorf("error migrating user table: %v", err)
	}

	// Add migration for comments table
	if err := db.migrateCommentsTable(); err != nil {
		return fmt.Errorf("error migrating comments table: %v", err)
	}

	// Create admin user if it doesn't exist
	if err := db.createAdminUser(); err != nil {
		return fmt.Errorf("error creating admin user: %v", err)
	}

	// Update existing admin user email if needed
	if err := db.updateAdminEmail(); err != nil {
		return fmt.Errorf("error updating admin email: %v", err)
	}

	// Insert default categories
	if err := db.insertDefaultCategories(); err != nil {
		return fmt.Errorf("error inserting default categories: %v", err)
	}

	return nil
}

// migrateUserTable adds new columns to existing user tables
func (db *DB) migrateUserTable() error {
	// Check if profile_picture column exists
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('users') 
		WHERE name='profile_picture'
	`).Scan(&columnExists)

	if err != nil {
		return err
	}

	if columnExists == 0 {
		// Add profile_picture column
		_, err = db.Exec("ALTER TABLE users ADD COLUMN profile_picture TEXT DEFAULT ''")
		if err != nil {
			return err
		}
	}

	// Check if signature column exists
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('users') 
		WHERE name='signature'
	`).Scan(&columnExists)

	if err != nil {
		return err
	}

	if columnExists == 0 {
		// Add signature column
		_, err = db.Exec("ALTER TABLE users ADD COLUMN signature TEXT DEFAULT ''")
		if err != nil {
			return err
		}
	}

	// Check if role column exists
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('users') 
		WHERE name='role'
	`).Scan(&columnExists)

	if err != nil {
		return err
	}

	if columnExists == 0 {
		// Add role column
		_, err = db.Exec("ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'user'")
		if err != nil {
			return err
		}
	}

	// Check if status column exists
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('users') 
		WHERE name='status'
	`).Scan(&columnExists)

	if err != nil {
		return err
	}

	if columnExists == 0 {
		// Add status column
		_, err = db.Exec("ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active'")
		if err != nil {
			return err
		}
	}

	return nil
}

// migrateCommentsTable adds new columns to existing comments tables
func (db *DB) migrateCommentsTable() error {
	// Check if parent_id column exists
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('comments') 
		WHERE name='parent_id'
	`).Scan(&columnExists)

	if err != nil {
		return err
	}

	if columnExists == 0 {
		// Add parent_id column
		_, err = db.Exec("ALTER TABLE comments ADD COLUMN parent_id INTEGER REFERENCES comments(id)")
		if err != nil {
			return err
		}
	}

	return nil
}

// createAdminUser creates the admin user if it doesn't exist
func (db *DB) createAdminUser() error {
	// Check if admin user already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ? OR email = ?", "admin", "admin@admin.com").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Admin user already exists
	}

	// Hash the admin password
	hashedPassword, err := auth.HashPassword("admin")
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %v", err)
	}

	// Create admin user
	query := "INSERT INTO users (username, email, password, role, status) VALUES (?, ?, ?, ?, ?)"
	_, err = db.Exec(query, "admin", "admin@admin.com", hashedPassword, "admin", "active")
	if err != nil {
		return err
	}

	return nil
}

// updateAdminEmail updates the admin user's email if it's still using the old format
func (db *DB) updateAdminEmail() error {
	// Check if admin user exists with old email format
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ? AND email = ?", "admin", "admin").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Update the admin user's email
		_, err = db.Exec("UPDATE users SET email = ? WHERE username = ? AND email = ?", "admin@admin.com", "admin", "admin")
		if err != nil {
			return err
		}
	}

	return nil
}

// insertDefaultCategories adds default categories for the literary forum
func (db *DB) insertDefaultCategories() error {
	categories := []struct {
		name        string
		description string
	}{
		{"General Discussion", "General book-related discussions and recommendations"},
		{"Fiction", "Discussions about fiction books and novels"},
		{"Non-Fiction", "Non-fiction books, biographies, and educational content"},
		{"Mystery & Thriller", "Mystery, thriller, and suspense novels"},
		{"Romance", "Romance novels and love stories"},
		{"Science Fiction & Fantasy", "Sci-fi, fantasy, and speculative fiction"},
		{"Classics", "Classic literature and timeless works"},
		{"Book Reviews", "Share and read book reviews"},
		{"Author Discussions", "Discussions about specific authors"},
		{"Book Club Picks", "Monthly book club selections and discussions"},
	}

	for _, cat := range categories {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM categories WHERE name = ?", cat.name).Scan(&count)
		if err != nil {
			return err
		}

		if count == 0 {
			_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, ?)", cat.name, cat.description)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// User operations
func (db *DB) CreateUser(user *models.User) error {
	query := "INSERT INTO users (username, email, password) VALUES (?, ?, ?)"
	result, err := db.Exec(query, user.Username, user.Email, user.Password)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = int(id)
	return nil
}

func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := "SELECT id, username, email, password, profile_picture, signature, role, status, created_at FROM users WHERE email = ?"
	err := db.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.ProfilePicture, &user.Signature, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) GetUserByID(id int) (*models.User, error) {
	user := &models.User{}
	query := "SELECT id, username, email, profile_picture, signature, role, status, created_at FROM users WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.ProfilePicture, &user.Signature, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	query := "SELECT id, username, email, profile_picture, signature, role, status, created_at FROM users WHERE username = ?"
	err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Email, &user.ProfilePicture, &user.Signature, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) UpdateUserProfile(userID int, profilePicture, signature string) error {
	query := "UPDATE users SET profile_picture = ?, signature = ? WHERE id = ?"
	_, err := db.Exec(query, profilePicture, signature, userID)
	return err
}

func (db *DB) CheckUserExists(email, username string) (bool, bool, error) {
	var emailCount, usernameCount int

	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&emailCount)
	if err != nil {
		return false, false, err
	}

	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&usernameCount)
	if err != nil {
		return false, false, err
	}

	return emailCount > 0, usernameCount > 0, nil
}

// Session operations
func (db *DB) CreateSession(session *models.Session) error {
	query := "INSERT INTO sessions (user_id, uuid, expires_at) VALUES (?, ?, ?)"
	result, err := db.Exec(query, session.UserID, session.UUID, session.ExpiresAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	session.ID = int(id)
	return nil
}

func (db *DB) GetSessionByUUID(uuid string) (*models.Session, error) {
	session := &models.Session{}
	query := "SELECT id, user_id, uuid, expires_at, created_at FROM sessions WHERE uuid = ? AND expires_at > ?"
	err := db.QueryRow(query, uuid, time.Now()).Scan(&session.ID, &session.UserID, &session.UUID, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (db *DB) DeleteSession(uuid string) error {
	query := "DELETE FROM sessions WHERE uuid = ?"
	_, err := db.Exec(query, uuid)
	return err
}

func (db *DB) CleanExpiredSessions() error {
	query := "DELETE FROM sessions WHERE expires_at < ?"
	_, err := db.Exec(query, time.Now())
	return err
}

// Category operations
func (db *DB) GetAllCategories() ([]models.Category, error) {
	query := "SELECT id, name, description, created_at FROM categories ORDER BY name"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		err := rows.Scan(&cat.ID, &cat.Name, &cat.Description, &cat.CreatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

func (db *DB) GetCategoryByID(id int) (*models.Category, error) {
	cat := &models.Category{}
	query := "SELECT id, name, description, created_at FROM categories WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&cat.ID, &cat.Name, &cat.Description, &cat.CreatedAt)
	if err != nil {
		return nil, err
	}
	return cat, nil
}

// Post operations
func (db *DB) CreatePost(post *models.Post) error {
	query := "INSERT INTO posts (title, content, user_id, category_id) VALUES (?, ?, ?, ?)"
	result, err := db.Exec(query, post.Title, post.Content, post.UserID, post.CategoryID)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	post.ID = int(id)
	return nil
}

func (db *DB) GetAllPosts() ([]models.Post, error) {
	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		ORDER BY p.created_at DESC
	`
	return db.executePosts(query)
}

func (db *DB) GetPostsByCategory(categoryID int) ([]models.Post, error) {
	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.category_id = ?
		ORDER BY p.created_at DESC
	`
	return db.executePostsWithArgs(query, categoryID)
}

func (db *DB) GetPostsByUser(userID int) ([]models.Post, error) {
	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.user_id = ?
		ORDER BY p.created_at DESC
	`
	return db.executePostsWithArgs(query, userID)
}

func (db *DB) GetLikedPostsByUser(userID int) ([]models.Post, error) {
	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE EXISTS (
			SELECT 1 FROM post_likes pl 
			WHERE pl.post_id = p.id AND pl.user_id = ? AND pl.is_like = 1
		)
		ORDER BY p.created_at DESC
	`
	return db.executePostsWithArgs(query, userID)
}
func (db *DB) GetPostByID(id int) (*models.Post, error) {
	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.id = ?
	`
	row := db.QueryRow(query, id)

	var post models.Post
	err := row.Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.CategoryID,
		&post.Username, &post.CategoryName, &post.CreatedAt, &post.UpdatedAt,
		&post.LikesCount, &post.DislikesCount, &post.CommentsCount)
	if err != nil {
		return nil, err
	}

	return &post, nil
}
func (db *DB) executePosts(query string) ([]models.Post, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.CategoryID,
			&post.Username, &post.CategoryName, &post.CreatedAt, &post.UpdatedAt,
			&post.LikesCount, &post.DislikesCount, &post.CommentsCount)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (db *DB) executePostsWithArgs(query string, args ...interface{}) ([]models.Post, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.CategoryID,
			&post.Username, &post.CategoryName, &post.CreatedAt, &post.UpdatedAt,
			&post.LikesCount, &post.DislikesCount, &post.CommentsCount)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// buildOrderClause builds the ORDER BY clause for sorting posts
func (db *DB) buildOrderClause(sortBy, sortOrder string) string {
	orderBy := "ORDER BY "

	switch sortBy {
	case "date":
		orderBy += "p.created_at"
	case "likes":
		orderBy += "likes_count"
	case "comments":
		orderBy += "comments_count"
	case "title":
		orderBy += "p.title"
	default:
		orderBy += "p.created_at"
	}

	if sortOrder == "asc" {
		orderBy += " ASC"
	} else {
		orderBy += " DESC"
	}

	return orderBy
}

// GetPostsWithSorting gets all posts with specified sorting
func (db *DB) GetPostsWithSorting(sortBy, sortOrder string) ([]models.Post, error) {
	orderClause := db.buildOrderClause(sortBy, sortOrder)

	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		` + orderClause

	return db.executePosts(query)
}

// GetPostsByCategoryWithSorting gets posts by category with specified sorting
func (db *DB) GetPostsByCategoryWithSorting(categoryID int, sortBy, sortOrder string) ([]models.Post, error) {
	orderClause := db.buildOrderClause(sortBy, sortOrder)

	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.category_id = ?
		` + orderClause

	return db.executePostsWithArgs(query, categoryID)
}

// GetPostsByUserWithSorting gets posts by user with specified sorting
func (db *DB) GetPostsByUserWithSorting(userID int, sortBy, sortOrder string) ([]models.Post, error) {
	orderClause := db.buildOrderClause(sortBy, sortOrder)

	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.user_id = ?
		` + orderClause

	return db.executePostsWithArgs(query, userID)
}

// GetLikedPostsByUserWithSorting gets liked posts by user with specified sorting
func (db *DB) GetLikedPostsByUserWithSorting(userID int, sortBy, sortOrder string) ([]models.Post, error) {
	orderClause := db.buildOrderClause(sortBy, sortOrder)

	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE EXISTS (
			SELECT 1 FROM post_likes pl 
			WHERE pl.post_id = p.id AND pl.user_id = ? AND pl.is_like = 1
		)
		` + orderClause

	return db.executePostsWithArgs(query, userID)
}

// GetPostsWithSuspendedFilterAndSorting gets posts with suspended filter and sorting
func (db *DB) GetPostsWithSuspendedFilterAndSorting(showSuspended bool, sortBy, sortOrder string) ([]models.Post, error) {
	orderClause := db.buildOrderClause(sortBy, sortOrder)

	baseQuery := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id`

	if !showSuspended {
		baseQuery += " WHERE u.status = 'active'"
	}

	query := baseQuery + " " + orderClause
	return db.executePosts(query)
}

// Comment operations
func (db *DB) CreateComment(comment *models.Comment) error {
	query := "INSERT INTO comments (content, user_id, post_id, parent_id) VALUES (?, ?, ?, ?)"
	result, err := db.Exec(query, comment.Content, comment.UserID, comment.PostID, comment.ParentID)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	comment.ID = int(id)
	return nil
}

func (db *DB) GetCommentsByPostID(postID int) ([]models.Comment, error) {
	query := `
		SELECT c.id, c.content, c.user_id, c.post_id, c.parent_id, u.username, c.created_at,
		       COALESCE(SUM(CASE WHEN cl.is_like = 1 THEN 1 ELSE 0 END), 0) as likes_count,
		       COALESCE(SUM(CASE WHEN cl.is_like = 0 THEN 1 ELSE 0 END), 0) as dislikes_count
		FROM comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN comment_likes cl ON c.id = cl.comment_id
		WHERE c.post_id = ?
		GROUP BY c.id, c.content, c.user_id, c.post_id, c.parent_id, u.username, c.created_at
		ORDER BY c.created_at ASC
	`
	rows, err := db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.UserID, &comment.PostID,
			&comment.ParentID, &comment.Username, &comment.CreatedAt, &comment.LikesCount, &comment.DislikesCount)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// Like operations
func (db *DB) LikePost(userID, postID int, isLike bool) error {
	// First, check if user already has a like/dislike on this post
	var existingLike sql.NullBool
	query := "SELECT is_like FROM post_likes WHERE user_id = ? AND post_id = ?"
	err := db.QueryRow(query, userID, postID).Scan(&existingLike)

	if err == sql.ErrNoRows {
		// No existing like, insert new one
		query = "INSERT INTO post_likes (user_id, post_id, is_like) VALUES (?, ?, ?)"
		_, err = db.Exec(query, userID, postID, isLike)
		return err
	} else if err != nil {
		return err
	}

	// Existing like found
	if existingLike.Valid && existingLike.Bool == isLike {
		// Same type of like, remove it
		query = "DELETE FROM post_likes WHERE user_id = ? AND post_id = ?"
		_, err = db.Exec(query, userID, postID)
		return err
	} else {
		// Different type of like, update it
		query = "UPDATE post_likes SET is_like = ? WHERE user_id = ? AND post_id = ?"
		_, err = db.Exec(query, isLike, userID, postID)
		return err
	}
}

func (db *DB) LikeComment(userID, commentID int, isLike bool) error {
	// First, check if user already has a like/dislike on this comment
	var existingLike sql.NullBool
	query := "SELECT is_like FROM comment_likes WHERE user_id = ? AND comment_id = ?"
	err := db.QueryRow(query, userID, commentID).Scan(&existingLike)

	if err == sql.ErrNoRows {
		// No existing like, insert new one
		query = "INSERT INTO comment_likes (user_id, comment_id, is_like) VALUES (?, ?, ?)"
		_, err = db.Exec(query, userID, commentID, isLike)
		return err
	} else if err != nil {
		return err
	}

	// Existing like found
	if existingLike.Valid && existingLike.Bool == isLike {
		// Same type of like, remove it
		query = "DELETE FROM comment_likes WHERE user_id = ? AND comment_id = ?"
		_, err = db.Exec(query, userID, commentID)
		return err
	} else {
		// Different type of like, update it
		query = "UPDATE comment_likes SET is_like = ? WHERE user_id = ? AND comment_id = ?"
		_, err = db.Exec(query, isLike, userID, commentID)
		return err
	}
}

func (db *DB) GetPostLikeStatus(userID, postID int) (bool, bool, error) {
	var isLike sql.NullBool
	query := "SELECT is_like FROM post_likes WHERE user_id = ? AND post_id = ?"
	err := db.QueryRow(query, userID, postID).Scan(&isLike)

	if err == sql.ErrNoRows {
		return false, false, nil // No like/dislike
	} else if err != nil {
		return false, false, err
	}

	if isLike.Valid {
		return isLike.Bool, !isLike.Bool, nil
	}

	return false, false, nil
}

func (db *DB) GetCommentLikeStatus(userID, commentID int) (bool, bool, error) {
	var isLike sql.NullBool
	query := "SELECT is_like FROM comment_likes WHERE user_id = ? AND comment_id = ?"
	err := db.QueryRow(query, userID, commentID).Scan(&isLike)

	if err == sql.ErrNoRows {
		return false, false, nil // No like/dislike
	} else if err != nil {
		return false, false, err
	}

	if isLike.Valid {
		return isLike.Bool, !isLike.Bool, nil
	}

	return false, false, nil
}

// Search operations
func (db *DB) SearchPosts(searchTerm string, limit int) ([]models.Post, error) {
	searchPattern := "%" + searchTerm + "%"
	query := `
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.title LIKE ? OR p.content LIKE ?
		ORDER BY p.created_at DESC
		LIMIT ?
	`
	return db.executePostsWithArgs(query, searchPattern, searchPattern, limit)
}

func (db *DB) SearchPostSuggestions(searchTerm string, limit int) ([]models.Post, error) {
	searchPattern := "%" + searchTerm + "%"
	query := `
		SELECT p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
		       p.created_at, p.updated_at,
		       0 as likes_count, 0 as dislikes_count, 0 as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.title LIKE ?
		ORDER BY p.created_at DESC
		LIMIT ?
	`
	return db.executePostsWithArgs(query, searchPattern, limit)
}

// DeleteUser deletes a user and all related data (posts, comments, likes, sessions)
// The deletion order is important due to foreign key constraints
func (db *DB) DeleteUser(userID int) error {
	// Start a transaction to ensure all deletions succeed or fail together
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// 1. Delete comment likes for comments on user's posts and user's comment likes
	_, err = tx.Exec(`
		DELETE FROM comment_likes 
		WHERE comment_id IN (
			SELECT c.id FROM comments c 
			JOIN posts p ON c.post_id = p.id 
			WHERE p.user_id = ?
		) OR user_id = ?
	`, userID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete comment likes: %v", err)
	}

	// 2. Delete post likes for user's posts and user's post likes
	_, err = tx.Exec(`
		DELETE FROM post_likes 
		WHERE post_id IN (
			SELECT id FROM posts WHERE user_id = ?
		) OR user_id = ?
	`, userID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete post likes: %v", err)
	}

	// 3. Delete comments on user's posts and user's comments
	_, err = tx.Exec(`
		DELETE FROM comments 
		WHERE post_id IN (
			SELECT id FROM posts WHERE user_id = ?
		) OR user_id = ?
	`, userID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete comments: %v", err)
	}

	// 4. Delete user's posts
	_, err = tx.Exec("DELETE FROM posts WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete posts: %v", err)
	}

	// 5. Delete user's sessions
	_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete sessions: %v", err)
	}

	// 6. Finally, delete the user
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// Admin operations
func (db *DB) GetAllUsers() ([]models.User, error) {
	query := `
		SELECT id, username, email, profile_picture, signature, role, status, created_at 
		FROM users 
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.ProfilePicture,
			&user.Signature, &user.Role, &user.Status, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// SuspendUser suspends a user (changes status to 'suspended')
func (db *DB) SuspendUser(userID int) error {
	query := "UPDATE users SET status = 'suspended' WHERE id = ? AND role != 'admin'"
	result, err := db.Exec(query, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or cannot suspend admin user")
	}

	return nil
}

// UnsuspendUser reactivates a suspended user (changes status to 'active')
func (db *DB) UnsuspendUser(userID int) error {
	query := "UPDATE users SET status = 'active' WHERE id = ?"
	_, err := db.Exec(query, userID)
	return err
}

// GetUserStats returns statistics about a user (posts, comments, likes)
func (db *DB) GetUserStats(userID int) (int, int, int, error) {
	var postsCount, commentsCount, likesReceived int

	// Count posts
	err := db.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", userID).Scan(&postsCount)
	if err != nil {
		return 0, 0, 0, err
	}

	// Count comments
	err = db.QueryRow("SELECT COUNT(*) FROM comments WHERE user_id = ?", userID).Scan(&commentsCount)
	if err != nil {
		return 0, 0, 0, err
	}

	// Count likes received on user's posts
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT p.id) FROM post_likes pl 
		JOIN posts p ON pl.post_id = p.id 
		WHERE p.user_id = ? AND pl.is_like = 1
	`, userID).Scan(&likesReceived)
	if err != nil {
		return 0, 0, 0, err
	}

	return postsCount, commentsCount, likesReceived, nil
}

// GetPostsWithSuspendedFilter gets posts, optionally filtering out suspended users' content
func (db *DB) GetPostsWithSuspendedFilter(showSuspended bool) ([]models.Post, error) {
	whereClause := ""
	if !showSuspended {
		whereClause = "WHERE u.status = 'active'"
	}

	query := fmt.Sprintf(`
		SELECT 
			p.id, p.title, p.content, p.user_id, p.category_id, u.username, c.name, 
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) as likes_count,
			(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 0) as dislikes_count,
			(SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id) as comments_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		%s
		ORDER BY p.created_at DESC
	`, whereClause)

	return db.executePosts(query)
}

// GetCommentsWithSuspendedFilter gets comments for a post, optionally filtering out suspended users' content
func (db *DB) GetCommentsWithSuspendedFilter(postID int, showSuspended bool) ([]models.Comment, error) {
	whereClause := "WHERE c.post_id = ?"
	args := []interface{}{postID}

	if !showSuspended {
		whereClause += " AND u.status = 'active'"
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.content, c.user_id, c.post_id, c.parent_id, u.username, c.created_at,
		       COALESCE(SUM(CASE WHEN cl.is_like = 1 THEN 1 ELSE 0 END), 0) as likes_count,
		       COALESCE(SUM(CASE WHEN cl.is_like = 0 THEN 1 ELSE 0 END), 0) as dislikes_count
		FROM comments c
		JOIN users u ON c.user_id = u.id
		LEFT JOIN comment_likes cl ON c.id = cl.comment_id
		%s
		GROUP BY c.id, c.content, c.user_id, c.post_id, c.parent_id, u.username, c.created_at
		ORDER BY c.created_at ASC
	`, whereClause)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.UserID, &comment.PostID,
			&comment.ParentID, &comment.Username, &comment.CreatedAt, &comment.LikesCount, &comment.DislikesCount)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, nil
}
