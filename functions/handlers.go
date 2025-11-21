package functions

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// handleComment validates and stores a new comment, then reloads the same post page.
func handleComment(w http.ResponseWriter, r *http.Request, data *CommentPageData, db *sql.DB, userID int) {
	if err := r.ParseForm(); err != nil {
		fmt.Println("Failed to parse comment form", err)
		RenderError(w, "please try later", 500)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	if err := isValidComment(content); err != nil {
		data.Error = err.Error()
		data.PrevContent = content
		ExecuteTemplate(w, "comments.html", data, 400)
		return
	}

	_, err := db.Exec(Insert_Comment, data.Post.Id, userID, content)
	if err != nil {
		fmt.Println("Failed to insert comment: %w", err)
		RenderError(w, errPleaseTryLater, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
}

// HandleReaction inserts, updates or removes a like/dislike for a post or comment.
func HandleReaction(db *sql.DB, userID, targetID int, target, reactionType string) error {
	isLike := (reactionType == "like")

	targetColumn := "post_id"
	if target == "comment" {
		targetColumn = "comment_id"
	}

	var reactionID int
	var existingLike bool

	query := "SELECT id, is_like FROM reaction WHERE user_id = ? AND " + targetColumn + " = ?"
	err := db.QueryRow(query, userID, targetID).Scan(&reactionID, &existingLike)

	switch {
	case err == sql.ErrNoRows:
		insert := "INSERT INTO reaction (user_id, " + targetColumn + ", is_like) VALUES (?, ?, ?)"
		_, err = db.Exec(insert, userID, targetID, isLike)
		return err

	case err != nil:
		return err

	default:

		if existingLike == isLike {
			_, err = db.Exec("DELETE FROM reaction WHERE id = ?", reactionID)
		} else {
			_, err = db.Exec("UPDATE reaction SET is_like = ? WHERE id = ?", isLike, reactionID)
		}

		return err
	}
}


// GetFilteredPosts retrieves posts based on the selected filter (mine, liked, or all) and category constraints.
func GetFilteredPosts(db *sql.DB, categories []string, UserId int, filter, storedToken string, data *HomePageData) ([]Post, error) {
	posts := []Post{}
	var rows *sql.Rows
	var err error
	filter = strings.ToLower(strings.TrimSpace(filter))
	guest := false

	if UserId < 1 {
		guest = true
	}

	if guest && (filter == "mine" || filter == "liked") {
		return nil, errors.New("redirect")
	}

	switch filter {
	case "mine": // only get the post created by the user
		data.Filter = filter
		rows, err = db.Query(Filter_Mine, UserId)

	case "liked": // only get posts liked by the user. Disliked posts won't be retrieved (we can change this later if we decide to filter by reacted posts)
		data.Filter = filter
		rows, err = db.Query(Filter_Liked, UserId)

	case "": // get all the posts
		rows, err = db.Query(No_Filter)
	default:
		return nil, errors.New("unknown filter")
	}

	if err != nil {
		return posts, fmt.Errorf("failed to int query for retrieving posts: %v", err)
	}

	defer rows.Close()
	allowed := map[string]bool{}

	for _, category := range categories {
		if strings.TrimSpace(category) == "" {
			continue
		}
		allowed[category] = true
	}

	for rows.Next() {
		var postId int

		err := rows.Scan(&postId)
		if err != nil {
			return nil, err
		}

		post, err := getPost(postId, db, UserId)
		if err != nil {
			return nil, err
		}

		if !guest {
			post.Token = storedToken
		}

		// if there is a category filter of the post is rejected ignore the post
		if len(allowed) > 0 && !Wanted(allowed, post) {
			continue
		}

		posts = append(posts, *post)
	}

	return posts, nil
}
