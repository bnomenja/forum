package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"forum/functions"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Println("usage: go run .")
		return
	}

	os.MkdirAll("db", 0o755)

	db, err := sql.Open("sqlite3", "db/forum.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	_, err = db.Exec(functions.Initialize)
	if err != nil {
		fmt.Println(err)
		return
	}

	database := &functions.Database{
		Db: db,
	}

	http.HandleFunc("/", database.Home)
	http.HandleFunc("/login", database.Login)
	http.HandleFunc("/register", database.Register)
	http.HandleFunc("/logout", database.Logout)
	http.HandleFunc("/create/post", database.CreatePost)
	http.HandleFunc("/posts/", database.CreateComment)
	http.HandleFunc("/reaction/", database.Reaction)
	http.HandleFunc("/statics/", functions.Css)

	fmt.Println("server started on http://localhost:8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}
