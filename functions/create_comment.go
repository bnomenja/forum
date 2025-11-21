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

	storedToken, userID, err1 := authenticateUser(r, database.Db)
	if userID == -1 { // something wrong happened
		RenderError(w, "please try later", 500)
		return
	}

	post, err := getPostWithDetails(postID, database.Db, storedToken, userID)
	if err != nil {
		if err.Error() == "post not found" {
			RenderError(w, "this post doesn't exist", 404)
			return
		}

		fmt.Println("Failed to retrieve post", err)
		RenderError(w, errPleaseTryLater, http.StatusInternalServerError)
		return
	}

	data := CommentPageData{Post: *post}

	if err1 == nil { // if user is logged
		data.Token = storedToken

		err := database.Db.QueryRow(Select_UserName, userID).Scan(&data.UserName)
		if err != nil {
			RenderError(w, "please try later", 500)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		if len(r.URL.RawQuery) > 0 {
			RenderError(w, "Method not allowed", 405)
			return
		}

		ExecuteTemplate(w, "comments.html", &data, 200)

	case http.MethodPost:
		if err1 != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if !ValidCSRF(r, storedToken) {
			RenderError(w, "Forbidden: CSRF Token Invalid", http.StatusForbidden)
			return
		}

		handleComment(w, r, &data, database.Db, userID)

	default:

		RenderError(w, errMethodNotAllowed, http.StatusMethodNotAllowed)
	}
}

func getPostWithDetails(postID int, db *sql.DB, storedToken string, userId int) (*Post, error) {
	post, err := getPost(postID, db, userId)
	if err != nil {
		return nil, err
	}

	post.Token = storedToken

	if err := getPostComments(post, db, storedToken, userId); err != nil {
		return nil, err
	}

	return post, nil
}
