package functions

import (
	"context"
	"database/sql"
	"net/http"
)

type Key string

func (database Database) Auth(handler http.HandlerFunc) http.HandlerFunc {
	db := database.Db

	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var userID int
		err = db.QueryRow(queryGetUserIDBySession, cookie.Value).Scan(&userID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			RenderError(w, "plesa try later", 500)
			return
		}

		var key Key = "user_id"

		ctx := context.WithValue(r.Context(), key, userID)

		handler(w, r.WithContext(ctx))
	}
}
