package functions

import (
	"database/sql"
	"fmt"
	"net/http"
)

func (database Database) CreateComment(w http.ResponseWriter, r *http.Request) {
	postID, err := extractPostID(r.URL.Path)
	if err != nil {
		RenderError(w, "this post doesn't exist", 404)
		return
	}

	post, err := getPostWithDetails(postID, database.Db)
	if err != nil {
		if err.Error() == "post not found" {
			RenderError(w, "this post doesn't exist", 404)
			return
		}

		fmt.Println("Failed to retrieve post", err)
		RenderError(w, errPleaseTryLater, http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ExecuteTemplate(w, "comments.html", &post, 200)
	case http.MethodPost:
		handleComment(w, r, post, database.Db)
	default:
		RenderError(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

func getPostWithDetails(postID int, db *sql.DB) (*Post, error) {
	post, err := getPost(postID, db)
	if err != nil {
		return nil, err
	}

	if err := getPostComments(post, db); err != nil {
		return nil, err
	}
	return post, nil
}
