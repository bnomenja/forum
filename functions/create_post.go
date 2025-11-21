package functions

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"
)

// CreatePost handles the /create/post route and displays or submits the post creation form.
func (database Database) CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/create/post" {
		RenderError(w, "Page not found", http.StatusNotFound)
		return
	}

	storedToken, userID, err := authenticateUser(r, database.Db)
	if userID == -1 {
		RenderError(w, "please try later", 500)
		return
	}

	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	db := database.Db

	switch r.Method {
	case http.MethodGet:
		if len(r.URL.RawQuery) > 0 {
			RenderError(w, "Method not allowed", 405)
			return
		}

		ExecuteTemplate(w, "post.html", PostPageData{CSRFToken: storedToken}, 200)

	case http.MethodPost:
		CreatePostHandler(w, r, db, userID, storedToken)

	default:
		RenderError(w, "Method not allowed", 405)
	}
}

// CreatePostHandler validates the form, checks CSRF, and inserts the post into the database.
func CreatePostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, userID int, storedToken string) {
	err := r.ParseForm()
	if err != nil {
		RenderError(w, "Please try later", 500)
		return
	}

	if !ValidCSRF(r, storedToken) {
		RenderError(w, "Forbidden: CSRF Token Invalid", http.StatusForbidden)
		return
	}

	post := MY_Post{
		Title:    r.FormValue("Title"),
		Content:  r.FormValue("Content"),
		Category: r.Form["Category"],
	}
	seen := map[string]bool{}
	for _, cat := range post.Category {
		if seen[cat] {
			PostPageData := PostPageData{
				ErrorMessege: errors.New("duplicated category"),
				Post:         post,
				CSRFToken:    storedToken,
			}
			ExecuteTemplate(w, "post.html", PostPageData, 400)
			return

		}
		seen[cat] = true
	}

	err = validate_post(&post)
	if err != nil {
		PostPageData := PostPageData{
			ErrorMessege: err,
			Post:         post,
			CSRFToken:    storedToken,
		}

		ExecuteTemplate(w, "post.html", PostPageData, 400)
		return
	}

	err = InsertPostToDB(w, db, &post, userID)
	if err != nil {
		fmt.Println("failed to insert post in database: ", err)
		RenderError(w, "please try later", 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// validate_post checks title, content, characters, and categories for correctness.
func validate_post(data *MY_Post) error {
	title := strings.TrimSpace(data.Title)
	contenue := strings.TrimSpace(data.Content)

	if title == "" {
		return errors.New("title is empty")
	}

	if contenue == "" {
		return errors.New("content is empty")
	}

	if len(title) > 150 {
		return errors.New("maximum number of title's character is 150")
	}

	if len(contenue) > 50000 {
		return errors.New(" maximum number of  content's character 50000")
	}

	if len(data.Category) == 0 {
		return errors.New("no categories")
	}

	for _, char := range title {
		if !unicode.IsPrint(char) {
			return fmt.Errorf("only printable characters are allowed")
		}
	}

	for _, char := range contenue {
		if !unicode.IsPrint(char) {
			return fmt.Errorf("only printable characters are allowed")
		}
	}

	allowed := map[string]bool{
		"Technology": true,
		"Science":    true,
		"Art":        true,
		"Gaming":     true,
		"Other":      true,
	}

	for _, catecategoryName := range data.Category {
		if _, exist := allowed[catecategoryName]; !exist {
			return errors.New("this category doesn't exist")
		}
	}

	return nil
}

// InsertPostToDB inserts a post and its categories inside a transaction.
func InsertPostToDB(w http.ResponseWriter, db *sql.DB, data *MY_Post, UserId int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	result, err := tx.Exec(Insert_Post, UserId, data.Title, data.Content)
	if err != nil {
		return err
	}

	PostID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	categories_id, err := getCategoriesId(data.Category, tx)
	if err != nil {
		return err
	}

	if err := insertInPost_Category(tx, int(PostID), categories_id); err != nil {
		return err
	}

	return nil
}

// ValidCSRF checks whether the submitted CSRF token matches the stored one.
func ValidCSRF(r *http.Request, storedToken string) bool {
	formToken := r.FormValue("csrf_token")

	if storedToken == "" || formToken == "" || storedToken != formToken {
		return false
	}

	return true
}

// getCategoriesId returns category IDs, creating new categories if they don't exist.
func getCategoriesId(Categories []string, tx *sql.Tx) ([]int, error) {
	categories_id := []int{}

	for _, category := range Categories {
		var categoryID int
		err := tx.QueryRow(Select_CategoryID, category).Scan(&categoryID)

		if err == sql.ErrNoRows {

			res, err1 := tx.Exec(Insert_Category, category)
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

// insertInPost_Category links the post with all its category IDs in post_category.
func insertInPost_Category(tx *sql.Tx, postId int, categories_id []int) error {
	stmt, err := tx.Prepare(INsert_Post_Category)
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
