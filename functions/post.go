package functions

import (
	"database/sql"
	"fmt"
	"net/http"
	"text/template"
)

func (database Database) CreatePosts(w http.ResponseWriter, r *http.Request) {
	db := database.Db

	if r.URL.Path != "/create/post" {
		RenderError(w, "page not found", 404)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tmpl, err := template.ParseFiles("templates/post.html")
		if err != nil {
			fmt.Println(err)
			return
		}

		err = tmpl.Execute(w, nil)
		if err != nil {
			fmt.Println(err)
		}

		return
	case http.MethodPost:
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		Session_ID := cookie.Value
		user_id := 0

		err = db.QueryRow("SELECT User_id FROM Session WHERE Id = ? AND Expires_at > CURRENT_TIMESTAMP", Session_ID).Scan(&user_id)
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user_name := ""
		err = db.QueryRow("SELECT Name FROM User WHERE Id =?", user_id).Scan(&user_name)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "please try later", 500)
			return
		}

		r.ParseForm()

		title := r.FormValue("title")
		content := r.FormValue("content")
		Categories := r.Form["categories"]

		tx, err := db.Begin()
		if err != nil {
			fmt.Println(err)
			RenderError(w, "failed to create the post", 500)
			return
		}

		res, err := tx.Exec("INSERT INTO Post(User_id, Title, Content) VALUES (?,?,?)", user_id, title, content)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "failed to create the post", 500)
			tx.Rollback()
			return
		}

		postId, err := res.LastInsertId()
		if err != nil {
			fmt.Println(err)
			RenderError(w, "failed to create the post", 500)
			tx.Rollback()
			return
		}

		categories_id := []int{}

		for _, category := range Categories {
			var categoryID int
			err := tx.QueryRow("SELECT Id FROM Category WHERE Type = ?", category).Scan(&categoryID)

			if err == sql.ErrNoRows {

				res, err := tx.Exec("INSERT INTO Category(Type) VALUES (?)", category)
				if err != nil {
					tx.Rollback()
					RenderError(w, "failed to create category", 500)
					return
				}
				id, _ := res.LastInsertId()
				categoryID = int(id)

			} else if err != nil {
				tx.Rollback()
				RenderError(w, "failed to create post", 500)
				return
			}

			categories_id = append(categories_id, categoryID)

		}

		stmt, err := tx.Prepare("INSERT INTO Post_Category(Post_id, Category_id) VALUES (?, ?)")
		if err != nil {
			fmt.Println(err)
			RenderError(w, "failed to create the post", 500)
			tx.Rollback()
			return
		}

		defer stmt.Close()

		for _, category_id := range categories_id {
			_, err := stmt.Exec(postId, category_id)
			if err != nil {
				fmt.Println(err)
				RenderError(w, "failed to create the post", 500)
				tx.Rollback()
				return
			}
		}

		if err := tx.Commit(); err != nil {
			fmt.Println(err)
			RenderError(w, "failed to create the post", 500)
			return
		}

	default:
		RenderError(w, "Method not allowed", 405)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
