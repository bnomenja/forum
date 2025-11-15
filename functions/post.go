package functions

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (database Database) CreatePosts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/create/post" {
		RenderError(w, "page not found", 404)
		return
	}

	user_id, err := database.authenticateUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ExecuteTemplate(w, "post.html", nil, 200)

	case http.MethodPost:
		code, err := HandleCreatePost(w, r, &database, user_id)
		if err != nil {
			RenderError(w, err.Error(), code)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		RenderError(w, "Method not allowed", 405)
	}
}

func HandleCreatePost(w http.ResponseWriter, r *http.Request, database *Database, user_id int) (int, error) {
	db := database.Db

	r.ParseForm()

	title := r.FormValue("title")
	content := r.FormValue("content")
	Categories := r.Form["categories"]

	if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
		return 400, errors.New("please fill all the required field when you create a post")
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("cannot initialize transaction", err)
		return 500, errors.New("failed to create the post")
	}

	res, err := tx.Exec("INSERT INTO Post(User_id, Title, Content) VALUES (?,?,?)", user_id, title, content)
	if err != nil {
		fmt.Println("failed to insert the post in his database", err)
		tx.Rollback()
		return 500, errors.New("failed to create the post")
	}

	postId, err := res.LastInsertId()
	if err != nil {
		fmt.Println("failed to get the post Id from his database", err)
		tx.Rollback()
		return 500, errors.New("failed to create the post")
	}

	categories_id, err := getCategoriesId(Categories, tx)
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		return 500, errors.New("failed to create the post")
	}

	if err := insertInPost_Category(tx, int(postId), categories_id); err != nil {
		fmt.Println(err)
		tx.Rollback()
		return 500, errors.New("failed to create the post")

	}

	return -1, nil
}
