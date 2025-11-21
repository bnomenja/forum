package functions

import "database/sql"

type Database struct {
	Db *sql.DB
}

type Reaction struct {
	UserId    int
	CommentId int
	PostId    int
	Islike    bool
}

type HomePageData struct {
	UserName string
	Filter   string
	Posts    []Post
	Token    string
}

type CommentPageData struct {
	UserName    string
	Error       string
	Post        Post
	Token       string
	PrevContent string
}

type Post struct {
	Id            int
	Title         string
	Content       string
	AuthorName    string
	AuthorId      int
	CreationDate  string
	Categories    []string
	CommentNumber int
	Comments      []Comment
	Likes         int
	Dislikes      int
	Liked         int // -1 : dislike;  0 : nothing; 1 : like
	Token         string
}

type RegisterData struct {
	Message  string
	Username string
	Email    string
}

type Comment struct {
	Id           int
	AuthorId     int
	AuthorName   string
	Content      string
	CreationDate string
	Likes        int
	Dislikes     int
	Token        string
	Liked        int
}

type MY_Post struct {
	Title    string
	Content  string
	Category []string
}

type PostPageData struct {
	ErrorMessege error
	Post         MY_Post
	CSRFToken    string
}

type ReactionData struct {
	ContentType     string
	ContentId       int
	ContentReaction string
}

type ErrorPage struct {
	Code    int
	Message string
}

type LoginData struct {
	Message  string
	Username string
}
