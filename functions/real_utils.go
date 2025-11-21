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

const (
	addCookie           = `INSERT INTO session(id, token, user_id, expire_at) VALUES (?, ?, ?, ?)`
	errPageNotFound     = "Page not found"
	errMethodNotAllowed = "Method not allowed"
	errPleaseTryLater   = "Please try later"
)

// post
func authenticateUser(r *http.Request, db *sql.DB) (string, int, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", 0, fmt.Errorf("session cookie not found: %w", err)
	}

	var userID int
	var storedToken string
	err = db.QueryRow(Select_UserID_and_Session, cookie.Value).Scan(&userID, &storedToken)
	if err == sql.ErrNoRows {
		return "", 0, fmt.Errorf("invalid or expired session: %w", err)
	}

	if err != nil {
		fmt.Println("cannot get the user ID", err)
		return "", -1, err
	}

	return storedToken, userID, nil
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

func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func SetNewSession(w http.ResponseWriter, db *sql.DB, userID int) error {
	sessionID, err1 := GenerateToken()
	csrf_token, err2 := GenerateToken()
	if err1 != nil || err2 != nil {
		return fmt.Errorf("failed to generate session")
	}

	expDate := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(addCookie, sessionID, csrf_token, userID, expDate)
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

func Redirect(target string, targetId int, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if target == "comment" { // you reacted on a comment -> redirect to comment page
		postId := 0
		err := db.QueryRow(Select_PostID, targetId).Scan(&postId)
		if err != nil {
			RenderError(w, "please try later", 500)
		}
		id := strconv.Itoa(postId)

		http.Redirect(w, r, "/posts/"+id, http.StatusSeeOther)

	} else { // you reacted on a post
		to := r.FormValue("redirect")

		switch to {
		case "home": // you reacted in the home page -> redirect to home
			http.Redirect(w, r, "/", http.StatusSeeOther)

		case "comment": // you reacted in comment page -> redirect to comment page
			id := strconv.Itoa(targetId)
			http.Redirect(w, r, "/posts/"+id, http.StatusSeeOther)

		default: // you modified the target
			RenderError(w, "we cannot redirect you there", 400)
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
		err = db.QueryRow(Verify_CommentID, commentId).Scan(&verification)
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
		err = db.QueryRow(Verify_PostID, postId).Scan(&verification)
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
func InitializeData(w http.ResponseWriter, r *http.Request, db *sql.DB) (string, HomePageData, int, error) {
	var data HomePageData
	var user_id int
	var token, user_name string

	cookie, err := r.Cookie("session")

	switch err {

	case nil: // user have a cookie
		Session_ID := cookie.Value

		err1 := db.QueryRow(Select_UserId_Csrf_UserName, Session_ID).Scan(&user_id, &token, &user_name)

		if err1 == sql.ErrNoRows { // the cookie is expired or invalid -> the user become a guest
			_, err2 := db.Exec(Delete_Session_by_ID, Session_ID)
			if err2 != nil {
				fmt.Println(err2)
				RenderError(w, "please try later", 500)
				return "", HomePageData{}, -1, err2
			}

			RemoveCookie(w)

			return "", HomePageData{}, -1, nil
		}

		if err1 != nil { // something wrong happened
			fmt.Println("a")
			fmt.Println(err1)
			RenderError(w, "please try later", 500)
			return "", HomePageData{}, -1, err1
		}

		data.UserName = user_name

	case http.ErrNoCookie: // user doesn't have a cookie -> user is a guest

	}

	return token, data, user_id, nil
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
	fmt.Println("categories: ", categories)
	allowed := map[string]bool{
		"Technology": true,
		"Science":    true,
		"Art":        true,
		"Gaming":     true,
		"Other":      true,
	}

	for _, category := range categories {
		if !allowed[(strings.TrimSpace(category))] {
			return false
		}
	}

	return true
}

func getPostCountData(post *Post, db *sql.DB) error {
	err := db.QueryRow(Select_Number, post.Id, post.Id, post.Id).Scan(&post.Likes, &post.Dislikes, &post.CommentNumber)
	if err != nil {
		return err
	}

	return nil
}

func getPostBasicInfo(postID int, db *sql.DB) (*Post, error) {
	post := &Post{Id: postID}
	var createdAt time.Time

	err := db.QueryRow(Select_Post_Basics, postID).Scan(&post.AuthorId, &post.Title, &post.Content, &createdAt, &post.AuthorName)
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
	rows, err := db.Query(Select_Categories, post.Id)
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

func getPostComments(post *Post, db *sql.DB, storedToken string, userID int) error {
	rows, err := db.Query(Select_Comment_Basics, post.Id)
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

		newcomment.Token = storedToken

		if userID > 0 {
			if err := getUserReactOnComments(&newcomment, db, userID); err != nil {
				return err
			}
		} else {
			newcomment.Liked = 0
		}
		newcomment.CreationDate = createdAt.Format("2006 Jan 2 15:04")

		post.Comments = append(post.Comments, newcomment)

	}

	return nil
}

func getUserReactOnComments(comment *Comment, db *sql.DB, UserID int) error {
	var liked bool
	err := db.QueryRow(Select_Reacted_On_Comment, comment.Id, UserID).Scan(&liked)
	if err == sql.ErrNoRows {
		comment.Liked = 0
		return nil
	}

	if err != nil {
		return err
	}

	if liked {
		comment.Liked = 1
	} else {
		comment.Liked = -1
	}

	return nil
}

func getPost(postId int, db *sql.DB, UserID int) (*Post, error) {
	post, err := getPostBasicInfo(postId, db)
	if err != nil {
		return nil, err
	}

	if err := getPostCategories(post, db); err != nil {
		return nil, err
	}

	if err := getPostCountData(post, db); err != nil {
		return nil, err
	}

	if UserID > 0 {
		if err := getUserReactOnPost(post, db, UserID); err != nil {
			return nil, err
		}
	} else {
		post.Liked = 0
	}

	return post, nil
}

func getUserReactOnPost(post *Post, db *sql.DB, UserID int) error {
	var liked bool
	err := db.QueryRow(Select_Reacted_On_Post, post.Id, UserID).Scan(&liked)
	if err == sql.ErrNoRows {
		post.Liked = 0
		return nil
	}

	if err != nil {
		return err
	}

	if liked {
		post.Liked = 1
	} else {
		post.Liked = -1
	}

	return nil
}

func Wanted(allowed map[string]bool, post *Post) bool {
	for _, postCategory := range post.Categories {
		if allowed[postCategory] {
			return true
		}
	}

	return false
}
