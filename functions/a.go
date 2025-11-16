package functions

import (
	"database/sql"
	"fmt"
)

func (database Database) GetPostWithDetails(postID int) (*Post, error) {
	var post Post
	db := database.Db

	// 1. Basic post info + author name in one query
	err := db.QueryRow(`
		SELECT p.Id, p.Title, p.Content, p.User_id, p.Created_at, u.Name
		FROM Post p
		JOIN User u ON u.Id = p.User_id
		WHERE p.Id = ?
	`, postID).Scan(
		&post.Id,
		&post.Title,
		&post.Content,
		&post.AuthorId,
		&post.CreationDate,
		&post.AuthorName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("failed to load post: %w", err)
	}

	// Prepare slices
	post.Categories = []string{}
	post.Comments = []Comment{}

	// 2. Load categories in one go
	catRows, err := db.Query(`
		SELECT c.Type
		FROM Category c
		JOIN Post_category pc ON pc.Category_id = c.Id
		WHERE pc.Post_id = ?
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to load categories: %w", err)
	}
	for catRows.Next() {
		var cat string
		if err := catRows.Scan(&cat); err != nil {
			return nil, err
		}
		post.Categories = append(post.Categories, cat)
	}
	catRows.Close()

	// 3. Load reactions count
	if err := db.QueryRow(`
		SELECT 
			SUM(CASE WHEN Is_like = 1 THEN 1 ELSE 0 END),
			SUM(CASE WHEN Is_like = 0 THEN 1 ELSE 0 END)
		FROM Reaction
		WHERE Post_id = ?
	`, postID).Scan(&post.Likes, &post.Dislikes); err != nil {
		return nil, fmt.Errorf("failed to load post reactions: %w", err)
	}

	// 4. Load comments + author names + reactions
	comRows, err := db.Query(`
		SELECT c.Id, c.User_id, u.Name, c.Content, c.Created_at,
			(SELECT COUNT(*) FROM Reaction r WHERE r.Comment_id = c.Id AND r.Is_like = 1),
			(SELECT COUNT(*) FROM Reaction r WHERE r.Comment_id = c.Id AND r.Is_like = 0)
		FROM Comment c
		JOIN User u ON u.Id = c.User_id
		WHERE c.Post_id = ?
		ORDER BY c.Created_at
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to load comments: %w", err)
	}

	for comRows.Next() {
		var com Comment
		if err := comRows.Scan(&com.Id, &com.AuthorId, &com.AuthorName,
			&com.Content, &com.CreationDate, &com.Likes, &com.Dislikes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		post.Comments = append(post.Comments, com)
	}
	comRows.Close()

	post.CommentNumber = len(post.Comments)

	return &post, nil
}
