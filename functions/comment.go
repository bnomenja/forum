package functions

import (
	"fmt"
	"net/http"
)

func (database Database) CreateComment(w http.ResponseWriter, r *http.Request) {
	postID, err := extractPostID(r.URL.Path)
	if err != nil {
		RenderError(w, "this post doesn't exist", 404)
		return
	}

	post, err := database.getPostWithDetails(postID)
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
		database.handleComment(w, r, post)
	default:
		RenderError(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}
