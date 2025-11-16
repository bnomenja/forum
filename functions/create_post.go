package functions

import (
	"errors"
	"fmt"
	"net/http"
)

func (database Database) CreatePosts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/create/post" {
		RenderError(w, "page not found", 404)
		return
	}

	user_id, err := authenticateUser(r, database.Db) // user need to be loged to create post
	if user_id == -1 {                               // something wrong happened
		RenderError(w, "please try later", 500)
		return
	}

	if err != nil { // the user is not loged
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ExecuteTemplate(w, "post.html", nil, 200) // just serving the create post page

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

	err := isValidPost(title, content, Categories)
	if err != nil {
		return 400, err
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
