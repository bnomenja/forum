package functions

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"
)

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
