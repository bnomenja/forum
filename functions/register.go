package functions

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

// Register handles GET and POST logic for the /register route.
func (database Database) Register(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/register" {
		RenderError(w, "Page not found", 404)
		return
	}

	switch r.Method {

	case http.MethodGet:
		if len(r.URL.RawQuery) > 0 {
			RenderError(w, "Method not allowed", 405)
			return
		}

		ExecuteTemplate(w, "register.html", nil, 200)

	case http.MethodPost:
		HandleRegister(w, r, database.Db)

	default:
		RenderError(w, "Method not allowed", 405)
	}
}

// HandleRegister processes user registration, validates data, inserts the user, and creates a session.
func HandleRegister(w http.ResponseWriter, r *http.Request, DB *sql.DB) {
	var data RegisterData

	data.Username = r.FormValue("username")
	password := r.FormValue("password")
	data.Email = r.FormValue("email")
	confirm_password := r.FormValue("confirm_password")

	err := IsValidCredentials(&data, password, confirm_password)
	if err != nil {
		data.Message = err.Error()
		ExecuteTemplate(w, "register.html", data, http.StatusBadRequest)
		return
	}

	//  check if the user exist
	var count int
	err = DB.QueryRow(Select_UserCount, data.Username, data.Email).Scan(&count)
	if err != nil {
		RenderError(w, "Please try later", 500)
		return
	}

	// exist deja ❌
	if count > 0 {
		data.Message = "❌ email or username already esist, please change"
		ExecuteTemplate(w, "register.html", data, http.StatusConflict)
		return
	}

	// password encryption
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Password encryption error:", err)
		RenderError(w, "Please try later", 500)
		return
	}

	// Enter in database
	res, err := DB.Exec(Insert_User, data.Username, data.Email, string(hashedPassword))
	if err != nil {
		fmt.Println("DB exec error:", err)
		RenderError(w, "Please try later", 500)
		return
	}

	// instead of asking the user to login again, we directly give a session and redirect him to home page
	// select the newUser'ID
	userID, err := res.LastInsertId()
	if err != nil {
		fmt.Println("failed to get new user ID:", err)
		RenderError(w, "Please try later", 500)
		return
	}

	// creating a new session
	err = SetNewSession(w, DB, int(userID))
	if err != nil {
		fmt.Println(err)
		RenderError(w, "Please try later", 500)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func IsValidCredentials(data *RegisterData, password, confirm_password string) error {
	// check email
	emailRegex := `^[a-zA-Z0-9._%+\-]{1,64}@[a-zA-Z0-9.\-]{1,255}\.[a-zA-Z]{2,10}$`
	re := regexp.MustCompile(emailRegex)
	if !re.MatchString(data.Email) {
		return errors.New("❌ Email format is invalid")
	}

	// check name
	userRegex := "^[A-Za-z][A-Za-z0-9_]{2,19}$"
	reU := regexp.MustCompile(userRegex)
	if !reU.MatchString(data.Username) {
		return errors.New("❌ Username format is invalid")
	}

	// check password
	if len(password) < 6 || len(password) > 20 {
		return errors.New("❌ Password must be between 6 and 20 characters")
	}

	if password != confirm_password {
		return errors.New("⚠️ Password is not matching")
	}

	return nil
}
