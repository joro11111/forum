package models

import (
	"time"
)

// User represents a registered user
type User struct {
	ID             int       `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Password       string    `json:"-"` // Don't include in JSON
	ProfilePicture string    `json:"profile_picture,omitempty"`
	Signature      string    `json:"signature,omitempty"`
	Role           string    `json:"role"`   // "user" or "admin"
	Status         string    `json:"status"` // "active" or "suspended"
	CreatedAt      time.Time `json:"created_at"`
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsSuspended checks if user is suspended
func (u *User) IsSuspended() bool {
	return u.Status == "suspended"
}

// Category represents a post category
type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Post represents a forum post
type Post struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	UserID        int       `json:"user_id"`
	CategoryID    int       `json:"category_id"`
	Username      string    `json:"username"`      // For display
	CategoryName  string    `json:"category_name"` // For display
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LikesCount    int       `json:"likes_count"`
	DislikesCount int       `json:"dislikes_count"`
	CommentsCount int       `json:"comments_count"`
}

// Comment represents a comment on a post
type Comment struct {
	ID            int       `json:"id"`
	Content       string    `json:"content"`
	UserID        int       `json:"user_id"`
	PostID        int       `json:"post_id"`
	Username      string    `json:"username"` // For display
	CreatedAt     time.Time `json:"created_at"`
	LikesCount    int       `json:"likes_count"`
	DislikesCount int       `json:"dislikes_count"`
}

// Session represents a user session
type Session struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	UUID      string    `json:"uuid"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// PostLike represents a like/dislike on a post
type PostLike struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	PostID    int       `json:"post_id"`
	IsLike    bool      `json:"is_like"` // true for like, false for dislike
	CreatedAt time.Time `json:"created_at"`
}

// CommentLike represents a like/dislike on a comment
type CommentLike struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	CommentID int       `json:"comment_id"`
	IsLike    bool      `json:"is_like"` // true for like, false for dislike
	CreatedAt time.Time `json:"created_at"`
}

// PostWithDetails represents a post with additional information for display
type PostWithDetails struct {
	Post
	UserCanLike  bool `json:"user_can_like"`
	UserLiked    bool `json:"user_liked"`
	UserDisliked bool `json:"user_disliked"`
}

// CommentWithDetails represents a comment with additional information for display
type CommentWithDetails struct {
	Comment
	UserCanLike  bool `json:"user_can_like"`
	UserLiked    bool `json:"user_liked"`
	UserDisliked bool `json:"user_disliked"`
}
