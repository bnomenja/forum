package functions

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
	queryGetPostDetails     = `SELECT User_id, Title, Content, Created_at FROM Post WHERE Id = ?`
	queryGetUserName        = `SELECT Name FROM User WHERE Id = ?`
	queryGetPostCategories  = `SELECT Category_id FROM Post_Category WHERE Post_id = ?`
	queryGetPostComments    = `SELECT Id, User_id, Content, Created_at FROM Comment WHERE Post_id = ? ORDER BY Created_at DESC`
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
	if err != nil {
		return nil, err
	}

	post.Id = postID

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

func isValidInput(input string) bool {
	if strings.TrimSpace(input) == "" {
		return false
	}

	for _, ch := range input {
		if !unicode.IsPrint(ch) {
			return false
		}
	}
	return true
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
		RenderError(w, "bad request", 400)
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

func GetAllCategories(db *sql.DB, post *Post) error {
	categories_id, err := GetCategoriesId(db, post)
	if err != nil {
		return err
	}

	for _, category_id := range categories_id {
		var category string
		err := db.QueryRow("SELECT Type FROM Category WHERE Id = ?", category_id).Scan(&category)
		if err != nil {
			return err
		}
		post.Categories = append(post.Categories, category)
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

func GetCategoriesId(db *sql.DB, post *Post) ([]int, error) {
	var Categories_id []int
	rows, err := db.Query("SELECT Category_id FROM Post_Category WHERE Post_id = ?", post.Id)
	if err != nil {
		return Categories_id, err
	}
	defer rows.Close()

	for rows.Next() {
		var Category_id int
		err := rows.Scan(&Category_id)
		if err != nil {
			return Categories_id, err
		}
		Categories_id = append(Categories_id, Category_id)
	}

	return Categories_id, nil
}

func InitializeData(w http.ResponseWriter, r *http.Request, db *sql.DB) (PageData, int, error) {
	var data PageData
	var user_id int

	cookie, err := r.Cookie("session")
	if err != nil {
		if err.Error() == "http: named cookie not present" {
		} else {
			fmt.Println("error getting cookie in home", err)
			RenderError(w, "please try later", 500)
			return PageData{}, -1, err
		}
	} else {
		Session_ID := cookie.Value

		err = db.QueryRow("SELECT User_id FROM Session WHERE Id = ? AND Expires_at > CURRENT_TIMESTAMP", Session_ID).Scan(&user_id)
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
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
	}

	return data, user_id, nil
}
