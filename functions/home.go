package functions

import (
	"database/sql"
	"fmt"
	"net/http"
	"text/template"
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

	db := database.Db
	data := PageData{}

	cookie, err := r.Cookie("session")
	user_id := 0

	if err != nil {
		if err.Error() == "http: named cookie not present" {
		} else {
			logError("error getting cookie in home", err)
			RenderError(w, "please try later", 500)
			return
		}
	} else {
		Session_ID := cookie.Value

		err = db.QueryRow("SELECT User_id FROM Session WHERE Id = ? AND Expires_at > CURRENT_TIMESTAMP", Session_ID).Scan(&user_id)
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user_name := ""
		err = db.QueryRow("SELECT Name FROM User WHERE Id = ?", user_id).Scan(&user_name)
		if err != nil {
			fmt.Println(err)
			RenderError(w, "please try later", 500)
			return
		}

		data.UserName = user_name
	}

	if err := r.ParseForm(); err != nil {
		logError("failed to parse form", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	filter := r.URL.Query().Get("filter")
	categories := r.Form["category"]

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		logError("failed to parse template", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	posts, err := GetPosts(db, categories, user_id, filter)
	if err != nil {
		logError("failed to load posts in home", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}

	data.Posts = posts

	err = tmpl.Execute(w, data)
	if err != nil {
		logError("failed to execute index template", err)
		RenderError(w, errPleaseTryLater, 500)
		return
	}
}

func GetPosts(db *sql.DB, categories []string, UserId int, filter string) ([]Post, error) {
	posts := []Post{}
	var rows *sql.Rows
	var err error

	switch filter {
	case "mine":
		rows, err = db.Query("SELECT * FROM Post WHERE User_id = ? ORDER BY Created_at DESC", UserId)
	case "liked":
		rows, err = db.Query(`
			SELECT p.*
			FROM Post p
			INNER JOIN Reaction r ON p.Id = r.Post_id
			WHERE r.User_id = ? AND r.Is_like = ?
			ORDER BY p.Created_at DESC
		`, UserId, true)
	default:
		rows, err = db.Query("SELECT * FROM Post ORDER BY Created_at DESC")
	}

	if err != nil {
		return posts, fmt.Errorf("failed to query posts: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var post Post
		err := rows.Scan(&post.Id, &post.AuthorId, &post.Title, &post.Content, &post.CreationDate)
		if err != nil {
			return posts, fmt.Errorf("failed to read rows: %v", err)
		}

		if err := GetAllCategories(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get categories of post %v", post.Id)
		}

		if len(categories) > 0 {
			matches := false
			for _, postCategory := range post.Categories {
				for _, filterCategory := range categories {
					if postCategory == filterCategory {
						matches = true
						break
					}
				}
				if matches {
					break
				}
			}
			if !matches {
				continue
			}
		}

		if err := GetAuthorName(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get author name for post %v", post.Id)
		}

		if err := GetCommentNumber(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get comment number for post %v", post.Id)
		}

		if err := GetReactionNumber(db, &post); err != nil {
			return posts, fmt.Errorf("failed to get reactions for post %v", post.Id)
		}

		posts = append(posts, post)
	}

	return posts, nil
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
