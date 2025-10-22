package functions

import (
	"fmt"
	"net/http"
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
		HandleCreatePost(w, r, &database, user_id)
		http.Redirect(w, r, "/", http.StatusSeeOther)

	default:
		RenderError(w, "Method not allowed", 405)
	}
}

func HandleCreatePost(w http.ResponseWriter, r *http.Request, database *Database, user_id int) {
	db := database.Db

	r.ParseForm()

	title := r.FormValue("title")
	content := r.FormValue("content")
	Categories := r.Form["categories"]

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("cannot initialize transaction", err)
		RenderError(w, "failed to create the post", 500)
		return
	}

	res, err := tx.Exec("INSERT INTO Post(User_id, Title, Content) VALUES (?,?,?)", user_id, title, content)
	if err != nil {
		fmt.Println("failed to insert the post in his database", err)
		RenderError(w, "failed to create the post", 500)
		tx.Rollback()
		return
	}

	postId, err := res.LastInsertId()
	if err != nil {
		fmt.Println("failed to get the post Id from his database", err)
		RenderError(w, "failed to create the post", 500)
		tx.Rollback()
		return
	}

	categories_id, err := getCategoriesId(Categories, tx)
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		RenderError(w, "failed to create category", 500)
		return
	}

	if err := insertInPost_Category(tx, int(postId), categories_id); err != nil {
		fmt.Println(err)
		RenderError(w, "failed to create the post", 500)
		tx.Rollback()
		return
	}
}
