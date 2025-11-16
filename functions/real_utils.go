package functions

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
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

	queryGetUserName       = `SELECT Name FROM User WHERE Id = ?`
	queryGetPostCategories = `SELECT Category_id FROM Post_Category WHERE Post_id = ?`
	queryGetPostComments   = `SELECT Id, User_id, Content, Created_at FROM Comment WHERE Post_id = ? ORDER BY Created_at DESC`
	queryGetCategoryType   = `SELECT Type FROM Category WHERE Id = ?`
	queryInsertComment     = `INSERT INTO Comment(Post_id, User_id, content) VALUES (?, ?, ?)`
	updateExpireDate       = `UPDATE Session SET Expires_at = ? WHERE Id = ?`
	addCookie              = `INSERT INTO Session(Id, User_id, Expires_at) VALUES (?, ?, ?)`
	errPageNotFound        = "Page not found"
	errMethodNotAllowed    = "Method not allowed"
	errPleaseTryLater      = "Please try later"
	Initialize             = `
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
	queryGetPostDetails = `
	SELECT p.User_id, p.Title, p.Content, p.Created_at , u.Name
	FROM Post p
	Join User u On u.Id = p.User_id
	WHERE p.Id = ?
	`

	queryGetcomment = `
	SELECT c.Id, c.User_Id, c.Content, c.Created_at, u.Name,
    (SELECT COUNT(*) FROM Reaction r WHERE r.Comment_id = c.Id AND r.Is_like = true),
    (SELECT COUNT(*) FROM Reaction r WHERE r.Comment_id = c.Id AND r.Is_like = false)
	FROM Comment c
	JOIN User u ON u.Id = c.User_Id
	WHERE c.Post_Id = ?
	ORDER BY c.Created_at DESC;
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

// post
func authenticateUser(r *http.Request, db *sql.DB) (int, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0, fmt.Errorf("session cookie not found: %w", err)
	}

	var userID int
	err = db.QueryRow(queryGetUserIDBySession, cookie.Value).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("invalid or expired session: %w", err)
	}

	if err != nil {
		fmt.Println("cannot get the user ID", err)
		return -1, err
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

func isValidComment(content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("comment must not be empty")
	}

	if len(content) > 1000 {
		return errors.New("maximum characters for a comment is 1000")
	}

	for _, ch := range content {
		if !unicode.IsPrint(ch) {
			return errors.New("only printable characters are allowed")
		}
	}

	return nil
}

func ExecuteTemplate(w http.ResponseWriter, filename string, data any, statutsCode int) {
	tmpl, err := template.ParseFiles("templates/" + filename)
	if err != nil {
		fmt.Printf("error while parsing %v: %v\n", filename, err)
		RenderError(w, "please try later", 500)
		return
	}

	var buff bytes.Buffer

	err1 := tmpl.Execute(&buff, data)
	if err1 != nil {
		fmt.Printf("error while executing %v: %v\n", filename, err1)
		RenderError(w, "please try later", 500)
		return
	}

	w.WriteHeader(statutsCode)

	_, err2 := buff.WriteTo(w)
	if err2 != nil {
		fmt.Printf("buffer error with %v: %v\n", filename, err2)
		RenderError(w, "please try later", 500)
		return
	}
}

func IsPrintable(data string) bool {
	for _, ch := range data {
		if !unicode.IsPrint(ch) {
			return false
		}
	}

	return true
}

func SetNewSession(w http.ResponseWriter, db *sql.DB, userID int) error {
	sessionID, err := GenerateSessionID()
	if err != nil {
		return fmt.Errorf("failed to generate session id: %v", err)
	}

	expDate := time.Now().Add(24 * time.Hour)

	_, err = db.Exec(addCookie, sessionID, userID, expDate)
	if err != nil {
		return fmt.Errorf("failed to add the session in database: %v", err)
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		Expires:  expDate,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, cookie)
	return nil
}

func IsValidCredential(name, email, password string) string {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
		return "You must fill all the fields"
	}

	if !IsPrintable(name) || !IsPrintable(email) || !IsPrintable(password) {
		return "Only printable characters are allowed as an input"
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return "Invalid email format"
	}

	if len(name) > 20 {
		return "Username must be less than 20 characters"
	}

	if len(password) < 8 {
		return "Password must be at least 8 characters long"
	}
	if len(password) > 64 {
		return "Password must be less than 64 characters"
	}

	haveNumber := false
	haveUpper := false
	havelower := false

	for _, ch := range password {
		if unicode.IsNumber(ch) {
			haveNumber = true
		}
		if unicode.IsUpper(ch) {
			haveUpper = true
		}
		if unicode.IsLower(ch) {
			havelower = true
		}

		if haveNumber && haveUpper && havelower {
			break
		}
	}

	if !haveNumber || !haveUpper || !havelower {
		return "Invalid Password (must contain number, upper case and lower case character)"
	}

	return ""
}

// GetCategoriesId will get the id of the categories passed as a parameter and return them as a slice of int
// It will add the category in the database if it hasn't been yet
// It also return an error if we failed to get an Id or or adding a category
func getCategoriesId(Categories []string, tx *sql.Tx) ([]int, error) {
	categories_id := []int{}

	for _, category := range Categories {
		var categoryID int
		err := tx.QueryRow("SELECT Id FROM Category WHERE Type = ?", category).Scan(&categoryID)

		if err == sql.ErrNoRows {

			res, err1 := tx.Exec("INSERT INTO Category(Type) VALUES (?)", category)
			if err1 != nil {
				return nil, err1
			}
			id, _ := res.LastInsertId()
			categoryID = int(id)

		} else if err != nil {
			return nil, err
		}

		categories_id = append(categories_id, categoryID)

	}
	return categories_id, nil
}

// insertInPost_Category will add the created post'Id and all his categories in post-cetgory table
func insertInPost_Category(tx *sql.Tx, postId int, categories_id []int) error {
	stmt, err := tx.Prepare("INSERT INTO Post_Category(Post_id, Category_id) VALUES (?, ?)")
	if err != nil {
	}

	defer stmt.Close()

	for _, category_id := range categories_id {
		_, err := stmt.Exec(postId, category_id)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func Redirect(target string, targetId int, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if target == "comment" {
		postId := 0
		db.QueryRow("SELECT Post_id FROM Comment WHERE Id = ?", targetId).Scan(&postId)
		Id := strconv.Itoa(postId)

		http.Redirect(w, r, "/posts/"+Id, http.StatusSeeOther)

	} else {
		to := r.FormValue("redirect")

		if to == "home" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else {
			Id := strconv.Itoa(targetId)
			http.Redirect(w, r, "/posts/"+Id, http.StatusSeeOther)
		}

	}
}

func getTargetId(target, id string, w http.ResponseWriter, db *sql.DB) int {
	targetId := -1

	switch target {
	case "comment":
		commentId, err := strconv.Atoi(id)
		if err != nil {
			RenderError(w, errPageNotFound, 404)
			return targetId
		}

		verification := ""
		err = db.QueryRow("SELECT Content FROM Comment WHERE ID =?", commentId).Scan(&verification)
		if err != nil {
			if err == sql.ErrNoRows {
				RenderError(w, "this comment doesn't exist", 404)
				return targetId
			}

			fmt.Println("error while confirming comment existance", err)
			RenderError(w, "you reacted on a non-existing comment", 400)
			return targetId
		}

		targetId = commentId

	case "post":
		postId, err := strconv.Atoi(id)
		if err != nil {
			RenderError(w, "you reacted on a non-existing post", 400)
			return targetId
		}

		verification := ""
		err = db.QueryRow("SELECT Title FROM Post WHERE ID =?", postId).Scan(&verification)
		if err != nil {
			if err == sql.ErrNoRows {
				RenderError(w, "this post doesn't exist", 404)
				return targetId
			}

			fmt.Println("error while confirming post existance", err)
			RenderError(w, errPageNotFound, 404)
			return targetId
		}

		targetId = postId

	default:
		fmt.Println("react to unknown")
		RenderError(w, "You can only react to post or comment", 400)
		return targetId
	}

	return targetId
}

func GetReactionNumber(db *sql.DB, post *Post) error {
	query := "SELECT COUNT(*) FROM Reaction WHERE Post_id = ? AND Is_like = ?"

	err := db.QueryRow(query, post.Id, true).Scan(&post.Likes)
	if err != nil {
		return err
	}

	err = db.QueryRow(query, post.Id, false).Scan(&post.Dislikes)
	if err != nil {
		return err
	}

	return nil
}

func GetCommentNumber(db *sql.DB, post *Post) error {
	err := db.QueryRow("SELECT COUNT(*) FROM Comment WHERE Post_id = ?", post.Id).Scan(&post.CommentNumber)
	if err != nil {
		return err
	}
	return nil
}

func GetAuthorName(db *sql.DB, post *Post) error {
	err := db.QueryRow("SELECT Name FROM User WHERE Id = ?", post.AuthorId).Scan(&post.AuthorName)
	if err != nil {
		return err
	}
	return nil
}

// Check if the user is loged to have his name and handle all errors related to that
func InitializeData(w http.ResponseWriter, r *http.Request, db *sql.DB) (PageData, int, error) {
	var data PageData
	var user_id int

	cookie, err := r.Cookie("session")

	switch err {

	case nil: // user have a cookie
		Session_ID := cookie.Value

		err = db.QueryRow("SELECT User_id FROM Session WHERE Id = ? AND Expires_at > CURRENT_TIMESTAMP", Session_ID).Scan(&user_id)

		if err == sql.ErrNoRows { // the cookie is expired or invalid -> the user become a guest
			_, err = db.Exec(queryDeleteSession, Session_ID)
			if err != nil {
				fmt.Println(err)
				RenderError(w, "please try later", 500)
				return PageData{}, -1, err
			}

			RemoveCookie(w)

			return data, user_id, nil
		}

		if err != nil {
			fmt.Println(err)
			RenderError(w, "please try later", 500)
			return PageData{}, -1, err
		}

		user_name := ""
		err = db.QueryRow("SELECT Name FROM User WHERE Id = ?", user_id).Scan(&user_name)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "please try later", 500)
			return PageData{}, -1, err
		}

		data.UserName = user_name

	case http.ErrNoCookie: // user doesn't have a cookie -> user is a guest

	default: // something wrong happened
		fmt.Println("error getting cookie in home", err)
		RenderError(w, "please try later", 500)
		return PageData{}, -1, err
	}

	return data, user_id, nil
}

func RemoveCookie(w http.ResponseWriter) {
	deleteCookie := &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   false,
	}

	http.SetCookie(w, deleteCookie)
}

func AreValidCategories(categories []string) bool {
	if len(categories) == 0 {
		return true
	}

	allowed := map[string]bool{"Technology": true, "Science": true, "Art": true, "Gaming": true, "Other": true}

	for _, category := range categories {
		if !allowed[(strings.TrimSpace(category))] {
			return false
		}
	}

	return true
}

func isValidPost(title, content string, categories []string) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	if title == "" || content == "" {
		return errors.New("please fill all the required field when you create a post")
	}

	if len(title) > 300 {
		return errors.New("title can only have 300 characters")
	}

	if len(content) > 50000 {
		return errors.New("content can only have 50000 characters")
	}

	for _, ch := range title {
		if !unicode.IsPrint(ch) {
			return errors.New("title can only contains printable characters")
		}
	}

	for _, ch := range content {
		if !unicode.IsPrint(ch) {
			return errors.New("content can only contains printable characters")
		}
	}

	allowed := map[string]bool{"Technology": true, "Science": true, "Art": true, "Gaming": true, "Other": true, "": true}

	for _, category := range categories {
		if !allowed[category] {
			return errors.New("you can only select the proposed categories")
		}
	}

	return nil
}

func getPosReactions(post *Post, db *sql.DB) error {
	queryGetPostReaction := `
	SELECT
		(SELECT COUNT(*) FROM Reaction r WHERE r.Post_id = ? AND r.Is_like = 1),
		(SELECT COUNT(*) FROM Reaction r WHERE r.Post_id = ? AND r.Is_like = 0)
	`
	err := db.QueryRow(queryGetPostReaction, post.Id, post.Id).Scan(&post.Likes, &post.Dislikes)
	if err != nil {
		return err
	}

	return nil
}

func getPostBasicInfo(postID int, db *sql.DB) (*Post, error) {
	var authorID int
	var authorName string
	var title, content string
	var createdAt time.Time

	err := db.QueryRow(queryGetPostDetails, postID).Scan(&authorID, &title, &content, &createdAt, &authorName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("failed to query post: %w", err)
	}

	post := &Post{
		Id:           postID,
		AuthorId:     authorID,
		AuthorName:   authorName,
		Title:        title,
		Content:      content,
		CreationDate: createdAt,
		Categories:   make([]string, 0),
		Comments:     make([]Comment, 0),
	}

	return post, nil
}

func getPostCategories(post *Post, db *sql.DB) error {
	query1 := `
	SELECT c.Type
	FROM Category c
	Join Post_Category pc ON pc.Category_id = c.Id
	WHERE Post_id = ?
	`
	rows, err := db.Query(query1, post.Id)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var category string

		err = rows.Scan(&category)
		if err != nil {
			return err
		}

		post.Categories = append(post.Categories, category)
	}

	return nil
}

func getPostComments(post *Post, db *sql.DB) error {
	rows, err := db.Query(queryGetcomment, post.Id)
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		newcomment := Comment{}

		err := rows.Scan(
			&newcomment.Id,
			&newcomment.AuthorId,
			&newcomment.Content,
			&newcomment.CreationDate,
			&newcomment.AuthorName,
			&newcomment.Likes,
			&newcomment.Dislikes,
		)
		if err != nil {
			return err
		}

		post.Comments = append(post.Comments, newcomment)

	}

	post.CommentNumber = len(post.Comments)

	return nil
}

func getPost(postId int, db *sql.DB) (*Post, error) {
	post, err := getPostBasicInfo(postId, db)
	if err != nil {
		return nil, err
	}

	if err := getPostCategories(post, db); err != nil {
		return nil, err
	}

	if err := getPosReactions(post, db); err != nil {
		return nil, err
	}

	return post, nil
}
