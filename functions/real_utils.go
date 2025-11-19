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
	CreationDate  string
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
	CreationDate string
	Likes        int
	Dislikes     int
}

const (
	queryDeleteSession      = `DELETE FROM session WHERE id = ?`
	queryGetUserIDBySession = `SELECT user_id FROM session WHERE id = ? AND expire_at > CURRENT_TIMESTAMP`
	queryInsertComment      = `INSERT INTO comment(post_id, user_id, content) VALUES (?, ?, ?)`
	addSession              = `INSERT INTO session(id, user_id, expire_at) VALUES (?, ?, ?)`
	errPageNotFound         = "Page not found"
	errMethodNotAllowed     = "Method not allowed"
	errPleaseTryLater       = "Please try later"

	Initialize = `
CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS session (
    id TEXT UNIQUE PRIMARY KEY,
    user_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expire_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES user(id)
);

CREATE TABLE IF NOT EXISTS post (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(id)
);

CREATE TABLE IF NOT EXISTS comment (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES post(id),
    FOREIGN KEY (user_id) REFERENCES user(id)
);

CREATE TABLE IF NOT EXISTS category (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS post_category (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    FOREIGN KEY (post_id) REFERENCES post(id),
    FOREIGN KEY (category_id) REFERENCES category(id),
    PRIMARY KEY (post_id, category_id)
);

CREATE TABLE IF NOT EXISTS reaction (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    post_id INTEGER,
    comment_id INTEGER,
    is_like BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(id),
    FOREIGN KEY (post_id) REFERENCES post(id),
    FOREIGN KEY (comment_id) REFERENCES comment(id),
    CONSTRAINT unique_reaction UNIQUE (user_id, post_id, comment_id)
);
`
	queryGetPostDetails = `
	SELECT p.user_id, p.title, p.content, p.created_at , u.name
	FROM post p
	Join user u On u.id = p.user_id
	WHERE p.id = ?
	`

	queryGetcomment = `
	SELECT c.id, c.user_Id, c.content, c.created_at, u.name,
    (SELECT COUNT(*) FROM reaction r WHERE r.comment_id = c.id AND r.is_like = true),
    (SELECT COUNT(*) FROM reaction r WHERE r.comment_id = c.id AND r.is_like = false)
	FROM comment c
	JOIN user u ON u.id = c.user_Id
	WHERE c.post_id = ?
	ORDER BY c.created_at DESC;
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
	var sessionID string
	var expDate time.Time

	for {
		sessionID, err := GenerateSessionID()
		if err != nil {
			return fmt.Errorf("failed to generate session id: %v", err)
		}

		expDate = time.Now().Add(24 * time.Hour)

		_, err = db.Exec(addSession, sessionID, userID, expDate)
		if err.Error() == "UNIQUE constraint failed: session.id" {
			continue
		}

		if err != nil {
			return fmt.Errorf("failed to add the session in database: %v", err)
		}

		break

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
// It also return an error if we failed to get an id or or adding a category
func getCategoriesId(Categories []string, tx *sql.Tx) ([]int, error) {
	categories_id := []int{}

	for _, category := range Categories {
		var categoryID int
		err := tx.QueryRow("SELECT id FROM category WHERE type = ?", category).Scan(&categoryID)

		if err == sql.ErrNoRows {

			res, err1 := tx.Exec("INSERT INTO category(type) VALUES (?)", category)
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

// insertInPost_Category will add the created post'id and all his categories in post-cetgory table
func insertInPost_Category(tx *sql.Tx, postId int, categories_id []int) error {
	stmt, err := tx.Prepare("INSERT INTO post_category(post_id, category_id) VALUES (?, ?)")
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
		db.QueryRow("SELECT post_id FROM comment WHERE id = ?", targetId).Scan(&postId)
		id := strconv.Itoa(postId)

		http.Redirect(w, r, "/posts/"+id, http.StatusSeeOther)

	} else {
		to := r.FormValue("redirect")

		if to == "home" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else {
			id := strconv.Itoa(targetId)
			http.Redirect(w, r, "/posts/"+id, http.StatusSeeOther)
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
		err = db.QueryRow("SELECT content FROM comment WHERE id =?", commentId).Scan(&verification)
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
		err = db.QueryRow("SELECT title FROM post WHERE id =?", postId).Scan(&verification)
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
	query := "SELECT COUNT(*) FROM reaction WHERE post_id = ? AND is_like = ?"

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
	err := db.QueryRow("SELECT COUNT(*) FROM comment WHERE post_id = ?", post.Id).Scan(&post.CommentNumber)
	if err != nil {
		return err
	}
	return nil
}

func GetAuthorName(db *sql.DB, post *Post) error {
	err := db.QueryRow("SELECT Name FROM User WHERE id = ?", post.AuthorId).Scan(&post.AuthorName)
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

		err = db.QueryRow("SELECT user_id FROM session WHERE id = ? AND expire_at > CURRENT_TIMESTAMP", Session_ID).Scan(&user_id)

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
		err = db.QueryRow("SELECT Name FROM User WHERE id = ?", user_id).Scan(&user_name)
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
		(SELECT COUNT(*) FROM reaction r WHERE r.post_id = ? AND r.is_like = true),
		(SELECT COUNT(*) FROM reaction r WHERE r.post_id = ? AND r.is_like = false)
	`
	err := db.QueryRow(queryGetPostReaction, post.Id, post.Id).Scan(&post.Likes, &post.Dislikes)
	if err != nil {
		return err
	}

	return nil
}

func getPostBasicInfo(postID int, db *sql.DB) (*Post, error) {
	post := &Post{Id: postID}
	var createdAt time.Time

	err := db.QueryRow(queryGetPostDetails, postID).Scan(&post.AuthorId, &post.Title, &post.Content, &createdAt, &post.AuthorName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("failed to query post: %w", err)
	}

	post.CreationDate = createdAt.Format("2006 Jan 2 15:04")

	return post, nil
}

func getPostCategories(post *Post, db *sql.DB) error {
	query1 := `
	SELECT c.Type
	FROM Category c
	Join Post_Category pc ON pc.Category_id = c.id
	WHERE post_id = ?
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
		createdAt := time.Time{}

		err := rows.Scan(
			&newcomment.Id,
			&newcomment.AuthorId,
			&newcomment.Content,
			&createdAt,
			&newcomment.AuthorName,
			&newcomment.Likes,
			&newcomment.Dislikes,
		)
		if err != nil {
			return err
		}

		newcomment.CreationDate = createdAt.Format("2006 Jan 2 15:04")

		post.Comments = append(post.Comments, newcomment)

	}

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

	if err := getPostCommentsNumber(post, db); err != nil {
		return nil, err
	}

	return post, nil
}

func getPostCommentsNumber(post *Post, db *sql.DB) error {
	err := db.QueryRow("SELECT COUNT(*) FROM comment  WHERE post_id = ? ", post.Id).Scan(&post.CommentNumber)
	if err != nil {
		return err
	}

	return nil
}
