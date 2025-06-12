# Literary Lions Forum

A web forum application for the Literary Lions book club to facilitate online discussions, book reviews, and literary engagement.

## Features

- **User Authentication**: Secure registration and login with session management
- **Posts & Comments**: Create and view posts with nested comments
- **Categories**: Organize posts by categories (specific books, genres, analyses, etc.)
- **Like/Dislike System**: Rate posts and comments
- **Post Filtering**: Filter by category, user's created posts, or liked posts
- **Secure**: Password encryption and UUID-based session management
- **Dockerized**: Easy deployment with Docker

## Technology Stack

- **Backend**: Go with standard library only
- **Database**: SQLite with go-sqlite3 driver
- **Frontend**: HTML/CSS templates (no JavaScript)
- **Authentication**: Session-based with cookies and UUIDs
- **Containerization**: Docker

## Entity Relationship Diagram (ERD)

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│    Users    │    │ Categories  │    │    Posts    │
├─────────────┤    ├─────────────┤    ├─────────────┤
│ id (PK)     │    │ id (PK)     │    │ id (PK)     │
│ username    │    │ name        │    │ title       │
│ email       │    │ description │    │ content     │
│ password    │    │ created_at  │    │ user_id(FK) │
│ created_at  │    └─────────────┘    │ category_id │
└─────────────┘                       │ created_at  │
       │                              │ updated_at  │
       │                              └─────────────┘
       │                                     │
       │    ┌─────────────┐                 │
       │    │  Sessions   │                 │
       │    ├─────────────┤                 │
       │    │ id (PK)     │                 │
       │    │ user_id(FK) │                 │
       │    │ uuid        │                 │
       │    │ expires_at  │                 │
       │    │ created_at  │                 │
       │    └─────────────┘                 │
       │                                    │
       │    ┌─────────────┐                 │
       └────┤  Comments   │─────────────────┘
            ├─────────────┤
            │ id (PK)     │
            │ content     │
            │ user_id(FK) │
            │ post_id(FK) │
            │ created_at  │
            └─────────────┘
                   │
    ┌─────────────┐│┌─────────────┐
    │ Post_Likes  │││Comment_Likes│
    ├─────────────┤││├─────────────┤
    │ id (PK)     │││ id (PK)     │
    │ user_id(FK) │││ user_id(FK) │
    │ post_id(FK) │││ comment_id  │
    │ is_like     │││ is_like     │
    │ created_at  │││ created_at  │
    └─────────────┘│└─────────────┘
                   │
                   └─ FK relationship

Relationships:
- Users 1:M Posts (one user can create many posts)
- Users 1:M Comments (one user can create many comments)
- Posts 1:M Comments (one post can have many comments)
- Categories 1:M Posts (one category can contain many posts)
- Users 1:M Sessions (one user can have multiple sessions)
- Users M:M Posts (through Post_Likes - many users can like many posts)
- Users M:M Comments (through Comment_Likes - many users can like many comments)
```

## Setup Instructions

### Prerequisites
- Go 1.24.3 or higher
- Docker (optional, for containerized deployment)

### Local Development

1. Clone the repository and navigate to the project directory:
```bash
cd literary-lions
```

2. Install dependencies:
```bash
go mod tidy
```

3. Run the application:
```bash
go run main.go
```

4. Open your browser and visit `http://localhost:8080`

### Docker Deployment

1. Build the Docker image:
```bash
docker build -t literary-lions-forum .
```

2. Run the container:
```bash
docker run -p 8080:8080 literary-lions-forum
```

## Usage

1. **Registration**: Create an account with email, username, and password
2. **Login**: Access your account using email and password
3. **Create Posts**: Share your literary thoughts with categories
4. **Comment**: Engage in discussions on posts
5. **Like/Dislike**: Express your opinion on posts and comments
6. **Filter**: Find posts by category, your created posts, or liked posts

## Database Schema

The application uses SQLite with the following tables:
- `users`: User account information
- `categories`: Discussion categories
- `posts`: Forum posts
- `comments`: Post comments
- `sessions`: User session management
- `post_likes`: Post like/dislike tracking
- `comment_likes`: Comment like/dislike tracking 