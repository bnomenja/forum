package functions

import (
	"fmt"
	"net/http"
)

func (database Database) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		RenderError(w, "page not found", 404)
		return
	}

	if r.Method != http.MethodGet {
		RenderError(w, "method not allowed", 405)
		return
	}

	data, user_id, err := InitializeData(w, r, database.Db)
	if err != nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("failed to parse form", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	filter := r.URL.Query().Get("filter")
	categories := r.Form["category"]

	if !AreValidCategories(categories) {
		RenderError(w, "unknown category", 400)
		return
	}

	posts, err := GetFilteredPosts(database.Db, categories, user_id, filter)
	if err != nil {
		fmt.Println("failed to load posts in home", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	data.Posts = posts

	ExecuteTemplate(w, "index.html", data, 200)
}
