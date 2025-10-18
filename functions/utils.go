package functions

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Database struct {
	Db *sql.DB
}

type Reaction struct {
	UserId    int
	CommentId int
	PostId    int
	Islike    bool
}
type PageData struct {
	UserName string
	Posts    []Post
}

type Post struct {
	Id            int
	Title         string
	Content       string
	AuthorName    string
	AuthorId      int
	CreationDate  time.Time
	Categories    []string
	CommentNumber int
	Comments      []Comment
	Likes         int
	Dislikes      int
}

type Comment struct {
	Id           int
	AuthorId     int
	AuthorName   string
	Content      string
	CreationDate time.Time
	Likes        int
	Dislikes     int
}

const (
	queryDeleteSession      = `DELETE FROM Session WHERE Id = ?`
	queryAddUser            = `INSERT INTO User (name, email, password) VALUES (?, ?, ?)`
	queryGetUserIDByEmail   = `SELECT id FROM User WHERE email = ?`
	queryGetUserIDBySession = `SELECT User_id FROM Session WHERE Id = ? AND Expires_at > CURRENT_TIMESTAMP`
	queryGetPostDetails     = `SELECT User_id, Title, Content, Created_at FROM Post WHERE Id = ?`
	queryGetUserName        = `SELECT Name FROM User WHERE Id = ?`
	queryGetPostCategories  = `SELECT Category_id FROM Post_Category WHERE Post_id = ?`
	queryGetPostComments    = `SELECT Id, User_id, Content, Created_at FROM Comment WHERE Post_id = ?`
	queryGetCategoryType    = `SELECT Type FROM Category WHERE Id = ?`
	queryInsertComment      = `INSERT INTO Comment(Post_id, User_id, content) VALUES (?, ?, ?)`
	updateExpireDate        = `UPDATE Session SET Expires_at = ? WHERE Id = ?`
	addCookie               = `INSERT INTO Session(Id, User_id, Expires_at) VALUES (?, ?, ?)`
	errPageNotFound         = "Page not found"
	errMethodNotAllowed     = "Method not allowed"
	errPleaseTryLater       = "Please try later"
	Initialize              = `
CREATE TABLE IF NOT EXISTS User (
    Id INTEGER PRIMARY KEY AUTOINCREMENT,
    Name TEXT UNIQUE NOT NULL,
    Email TEXT UNIQUE NOT NULL,
    Password TEXT NOT NULL,
    Created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS Session (
    Id TEXT PRIMARY KEY,
    User_id INTEGER NOT NULL,
    Created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    Expires_at DATETIME,
    FOREIGN KEY (User_id) REFERENCES User(Id)
);

CREATE TABLE IF NOT EXISTS Post (
    Id INTEGER PRIMARY KEY AUTOINCREMENT,
    User_id INTEGER NOT NULL,
    Title TEXT NOT NULL,
    Content TEXT NOT NULL,
    Created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (User_id) REFERENCES User(Id)
);

CREATE TABLE IF NOT EXISTS Comment (
    Id INTEGER PRIMARY KEY AUTOINCREMENT,
    Post_id INTEGER NOT NULL,
    User_id INTEGER NOT NULL,
    Content TEXT NOT NULL,
    Created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (Post_id) REFERENCES Post(Id),
    FOREIGN KEY (User_id) REFERENCES User(Id)
);

CREATE TABLE IF NOT EXISTS Category (
    Id INTEGER PRIMARY KEY AUTOINCREMENT,
    Type TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS Post_Category (
    Post_id INTEGER NOT NULL,
    Category_id INTEGER NOT NULL,
    FOREIGN KEY (Post_id) REFERENCES Post(Id),
    FOREIGN KEY (Category_id) REFERENCES Category(Id),
    PRIMARY KEY (Post_id, Category_id)
);

CREATE TABLE IF NOT EXISTS Reaction (
    Id INTEGER PRIMARY KEY AUTOINCREMENT,
    User_id INTEGER NOT NULL,
    Post_id INTEGER,
    Comment_id INTEGER,
    Is_like BOOLEAN NOT NULL,
    Created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (User_id) REFERENCES User(Id),
    FOREIGN KEY (Post_id) REFERENCES Post(Id),
    FOREIGN KEY (Comment_id) REFERENCES Comment(Id),
    CONSTRAINT unique_reaction UNIQUE (User_id, Post_id, Comment_id)
);
`
)

func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func SetNewExpireDate(w http.ResponseWriter, db *sql.DB, user_id int, session_id string) error {
	newExp := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(updateExpireDate, newExp, user_id)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    session_id,
		Path:     "/",
		Expires:  newExp,
		HttpOnly: true,
		Secure:   false,
	}

	http.SetCookie(w, cookie)

	return nil
}

// register

func logError(context string, err error) {
	fmt.Printf("[ERROR] %s: %v\n", context, err)
}

// post
func (database Database) authenticateUser(r *http.Request) (int, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0, fmt.Errorf("session cookie not found: %w", err)
	}

	var userID int
	err = database.Db.QueryRow(queryGetUserIDBySession, cookie.Value).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("invalid or expired session: %w", err)
	}

	return userID, nil
}

// comments

func extractPostID(path string) (int, error) {
	id := strings.TrimPrefix(path, "/posts/")
	if id == "" {
		return 0, fmt.Errorf("missing post ID")
	}

	postID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("invalid post ID: %w", err)
	}

	return postID, nil
}

func (database Database) getPostWithDetails(postID int) (*Post, error) {
	post, err := database.getPostBasicInfo(postID)
	post.Id = postID
	if err != nil {
		return nil, err
	}

	if err := database.getPostAuthor(post, post.AuthorId); err != nil {
		return nil, err
	}

	if err := database.getPostCategories(post, postID); err != nil {
		return nil, err
	}

	if err := database.getPostComments(post, postID); err != nil {
		return nil, err
	}

	post.CommentNumber = len(post.Comments)

	if err := database.Db.QueryRow("SELECT COUNT (*) FROM Reaction WHERE  Post_id = ? AND Is_like = ?", post.Id, true).Scan(&post.Likes); err != nil {
		return nil, fmt.Errorf("failed to get comment likes number: %w", err)
	}

	if err := database.Db.QueryRow("SELECT COUNT (*) FROM Reaction WHERE  Post_id = ? AND Is_like = ?", post.Id, false).Scan(&post.Dislikes); err != nil {
		return nil, fmt.Errorf("failed to get comment dislikes number: %w", err)
	}

	return post, nil
}

func (database Database) getPostBasicInfo(postID int) (*Post, error) {
	var authorID int
	var title, content string
	var createdAt time.Time

	err := database.Db.QueryRow(queryGetPostDetails, postID).Scan(&authorID, &title, &content, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("failed to query post: %w", err)
	}

	post := &Post{
		Title:        title,
		Content:      content,
		AuthorId:     authorID,
		CreationDate: createdAt,
		Categories:   make([]string, 0),
		Comments:     make([]Comment, 0),
	}

	return post, nil
}

func (database Database) getPostAuthor(post *Post, authorID int) error {
	err := database.Db.QueryRow(queryGetUserName, authorID).Scan(&post.AuthorName)
	if err != nil {
		return fmt.Errorf("failed to get author name: %w", err)
	}
	return nil
}

func (database Database) getPostCategories(post *Post, postID int) error {
	rows, err := database.Db.Query(queryGetPostCategories, postID)
	if err != nil {
		return fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	categoryIDs := make([]int, 0)
	for rows.Next() {
		var categoryID int
		if err := rows.Scan(&categoryID); err != nil {
			return fmt.Errorf("failed to scan category ID: %w", err)
		}
		categoryIDs = append(categoryIDs, categoryID)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating categories: %w", err)
	}

	return database.getCategoryNames(post, categoryIDs)
}

func (database Database) getCategoryNames(post *Post, categoryIDs []int) error {
	if len(categoryIDs) == 0 {
		return nil
	}

	stmt, err := database.Db.Prepare(queryGetCategoryType)
	if err != nil {
		return fmt.Errorf("failed to prepare category statement: %w", err)
	}
	defer stmt.Close()

	for _, categoryID := range categoryIDs {
		var categoryType string
		if err := stmt.QueryRow(categoryID).Scan(&categoryType); err != nil {
			return fmt.Errorf("failed to get category type for ID %d: %w", categoryID, err)
		}
		post.Categories = append(post.Categories, categoryType)
	}

	return nil
}

func (database Database) getPostComments(post *Post, postID int) error {
	rows, err := database.Db.Query(queryGetPostComments, postID)
	if err != nil {
		return fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		comment, err := database.scanComment(rows)
		if err != nil {
			return err
		}
		post.Comments = append(post.Comments, comment)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating comments: %w", err)
	}

	return nil
}

func (database Database) scanComment(rows *sql.Rows) (Comment, error) {
	var commentId int
	var userID int
	var content string
	var createdAt time.Time

	if err := rows.Scan(&commentId, &userID, &content, &createdAt); err != nil {
		return Comment{}, fmt.Errorf("failed to scan comment: %w", err)
	}

	var authorName string
	if err := database.Db.QueryRow(queryGetUserName, userID).Scan(&authorName); err != nil {
		return Comment{}, fmt.Errorf("failed to get comment author name: %w", err)
	}

	likes := 0
	dislikes := 0
	if err := database.Db.QueryRow("SELECT COUNT(*) FROM Reaction WHERE Comment_id = ? AND Is_like = ?", commentId, true).Scan(&likes); err != nil {
		return Comment{}, fmt.Errorf("failed to get comment likes number: %w", err)
	}

	if err := database.Db.QueryRow("SELECT COUNT(*) FROM Reaction WHERE Comment_id = ? AND Is_like = ?", commentId, false).Scan(&dislikes); err != nil {
		return Comment{}, fmt.Errorf("failed to get comment dislikes number: %w", err)
	}

	return Comment{
		Id:           commentId,
		AuthorId:     userID,
		AuthorName:   authorName,
		Content:      content,
		CreationDate: createdAt,
		Likes:        likes,
		Dislikes:     dislikes,
	}, nil
}

func (database Database) handlePostRetrievalError(w http.ResponseWriter, err error) {
	if err.Error() == "post not found" {
		RenderError(w, errPageNotFound, http.StatusNotFound)
		return
	}
	logError("Failed to retrieve post", err)
	RenderError(w, errPleaseTryLater, http.StatusInternalServerError)
}

func (database Database) insertComment(postID, userID int, content string) error {
	_, err := database.Db.Exec(queryInsertComment, postID, userID, content)
	if err != nil {
		return fmt.Errorf("failed to execute insert: %w", err)
	}
	return nil
}
