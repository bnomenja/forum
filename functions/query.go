package functions

// for creating all tables and rows
const Initialize = `
CREATE TABLE IF NOT EXISTS user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY,
	token TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expire_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES user(id)
);

CREATE TABLE IF NOT EXISTS post (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(id)
);

CREATE TABLE IF NOT EXISTS comment (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES post(id),
    FOREIGN KEY (user_id) REFERENCES user(id)
);

CREATE TABLE IF NOT EXISTS category (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS post_category (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    FOREIGN KEY (post_id) REFERENCES post(id),
    FOREIGN KEY (category_id) REFERENCES category(id),
    PRIMARY KEY (post_id, category_id)
);

CREATE TABLE IF NOT EXISTS reaction (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    post_id INTEGER,
    comment_id INTEGER,
    is_like BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(id),
    FOREIGN KEY (post_id) REFERENCES post(id),
    FOREIGN KEY (comment_id) REFERENCES comment(id),
    CONSTRAINT unique_reaction UNIQUE (user_id, post_id, comment_id)
);
`

// for register, login 	and logout
const (
	Insert_User          = `INSERT INTO user (name, email, password) VALUES (?, ?, ?)`
	Select_UserCount     = `SELECT COUNT(*) FROM user WHERE name = ? OR email = ?`
	Select_UserID_and_Pw = `SELECT id, password FROM user WHERE name = ?`
	Delete_User_Session  = `DELETE FROM session where user_id=? `
	Select_SessionID     = `SELECT id FROM session WHERE user_id = ?`
	Delete_Session_by_ID = `DELETE FROM session WHERE id = ?`
)

// for create post
const (
	Insert_Post          = `INSERT INTO post (user_id, title, content) VALUES(?,?,?)`
	Insert_Category      = `INSERT INTO category(type) VALUES (?)`
	Select_CategoryID    = `SELECT id FROM category WHERE type = ?`
	INsert_Post_Category = `INSERT INTO post_category(post_id, category_id) VALUES (?, ?)`
)

// for retrieving post Data
const (
	Select_Post_Basics = `
	SELECT p.user_id, p.title, p.content, p.created_at , u.name
	FROM post p
	Join user u On u.id = p.user_id
	WHERE p.id = ?
	`

	Select_Categories = `
	SELECT c.Type
	FROM Category c
	Join Post_Category pc ON pc.Category_id = c.id
	WHERE post_id = ?
	`
	Select_Number = `
	SELECT
		(SELECT COUNT(*) FROM reaction r WHERE r.post_id = ? AND r.is_like = true),
		(SELECT COUNT(*) FROM reaction r WHERE r.post_id = ? AND r.is_like = false),
		(SELECT COUNT(*) FROM comment c WHERE c.post_id = ? )
	`

	Select_Reacted_On_Post = `SELECT is_like FROM reaction WHERE post_id = ? AND user_id = ?`
)

// for retrieving comment Data
const (
	Select_Comment_Basics = `
	SELECT c.id, c.user_Id, c.content, c.created_at, u.name,
    (SELECT COUNT(*) FROM reaction r WHERE r.comment_id = c.id AND r.is_like = true),
    (SELECT COUNT(*) FROM reaction r WHERE r.comment_id = c.id AND r.is_like = false)
	FROM comment c
	JOIN user u ON u.id = c.user_Id
	WHERE c.post_id = ?
	ORDER BY c.created_at DESC
`
	Select_Reacted_On_Comment = `SELECT is_like FROM reaction WHERE comment_id = ? AND user_id = ?`
)

// for create Comment
const Insert_Comment = `INSERT INTO comment(post_id, user_id, content) VALUES (?, ?, ?)`

// for home
const (
	Select_UserId_Csrf_UserName = `
	SELECT s.user_id, s.token, u.name
	FROM session s
	JOIN user u ON u.id = s.user_id
	 WHERE s.id = ? AND expire_at > CURRENT_TIMESTAMP
	`
	Filter_Liked = `
	SELECT p.id
	FROM post p
	JOIN reaction r ON p.id = r.post_id
	WHERE r.user_id = ? AND r.is_like = true
	ORDER BY p.created_at DESC
	`
	Filter_Mine = `SELECT id FROM post WHERE user_id = ? ORDER BY created_at DESC`
	No_Filter   = `SELECT id FROM post ORDER BY created_at DESC`
)

// for reaction
const (
	Verify_PostID    = `SELECT title FROM post WHERE id =?`
	Verify_CommentID = `SELECT content FROM comment WHERE id =?`
	// reaction have other query but they are dynamics
)

// for utils
const (
	Select_UserID_and_Session = `SELECT user_id, token FROM session WHERE id = ? AND expire_at > CURRENT_TIMESTAMP`
	Select_PostID             = `SELECT post_id FROM comment WHERE id = ?`
	Select_UserName           = `SELECT name FROM user WHERE id = ?`
)
