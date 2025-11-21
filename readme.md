# Forum

A web forum application built with Go and SQLite that enables user communication through posts and comments with category filtering and engagement features.

## Features

### Authentication
- User registration with email, username, and password
- Login session management using cookies
- Session expiration handling
- Password encryption with bcrypt

### Posts & Comments
- Create posts with associated categories
- Comment on posts
- View posts and comments (available to all visitors)
- Only registered users can create content

### Engagement System
- Like and dislike posts
- Like and dislike comments
- View like/dislike counts (visible to all users)
- Only registered users can interact

### Filtering
- Filter posts by categories
- Filter by user's created posts (registered users only)
- Filter by user's liked posts (registered users only)

## Tech Stack

- **Backend**: Go (Golang)
- **Database**: SQLite
- **Frontend**: HTML, CSS (vanilla)
- **Containerization**: Docker

## Project Structure

```
forum/
├── assets/
│   └── icons/
├── db/
│   └── forum.db
├── functions/
│   ├── create_comment.go
│   ├── create_post.go
│   ├── error.go
│   ├── handlers.go
│   ├── home.go
│   ├── login.go
│   ├── logout.go
│   ├── query.go
│   ├── reaction.go
│   ├── real_utils.go
│   ├── register.go
│   ├── serve_css.go
│   └── struct.go
├── statics/
│   ├── comment.css
│   ├── error.css
│   ├── index.css
│   ├── login&register.css
│   └── post.css.
├── templates/
│   ├── comments.css
│   ├── error.css
│   ├── index.css
│   ├── register.html
│   ├── login.html
│   └── post.html
├── go.mod
├── go.sum
├── main.go
└── Dockerfile
```

## Installation

### Prerequisites
- Docker installed on your system

### Running with Docker

1. Clone the repository:
```bash
git clone https://learn.zone01oujda.ma/git/wkhlifi/forum.git
cd forum
```

2. Build the Docker image:
```bash
docker build -t forum .
```

3. Run the container:
```bash
docker run -p 8080:8080 forum
```

4. Access the application at `http://localhost:8080`

## Database Schema

The application uses SQLite with the following main tables:
- **users**: User credentials and information
- **posts**: Forum posts with category associations
- **comments**: Comments on posts
- **likes**: Like/dislike records for posts and comments
- **sessions**: Active user sessions
- **categories**: Post categories

## Usage

### Registration
1. Navigate to the registration page
2. Provide email, username, and password
3. Submit to create your account

### Login
1. Enter your credentials on the login page
2. Session cookie will be created upon successful login
3. Cookie expires after a set duration

### Creating Posts
1. Log in to your account
2. Navigate to create post
3. Write your content and select categories
4. Submit to publish

### Interacting with Content
- **View**: All users can view posts and comments
- **Comment**: Registered users can add comments
- **Like/Dislike**: Registered users can engage with posts and comments
- **Filter**: Use category filters or personal filters (created/liked posts)

## Error Handling

The application handles:
- HTTP status errors
- Database connection errors
- Invalid user input
- Duplicate email registration
- Invalid login credentials
- Session expiration
- Unauthorized access attempts

## Security

- Passwords are encrypted using bcrypt
- Session management with secure cookies
- SQL injection prevention through prepared statements
- Input validation and sanitization

## Team

- **BEMAMORY Nomenjanahary Luciano Loic** (bnomenja)
- **Walid Khlifi** (wkhlifi)
- **Mohamed Nouri** (mohnouri)
- **Othmane Benmbarek** (obenmbar)

## License

This project is part of the Zone01 Oujda curriculum.

## Acknowledgments

Built as part of the Zone01 Oujda web development track to learn:
- Web fundamentals (HTML, HTTP, Sessions, Cookies)
- Docker containerization
- SQL database manipulation
- Go web application development
- Security best practices