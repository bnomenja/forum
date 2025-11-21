package functions

import (
	"net/http"
	"os"
)

// ServeCss serves CSS/static files securely and blocks directory access.
func ServeCss(w http.ResponseWriter, r *http.Request) {
	fileinfo, err := os.Stat(r.URL.Path[1:])
	if err != nil {
		RenderError(w, "page not found", http.StatusNotFound)
		return
	}

	if fileinfo.IsDir() {
		RenderError(w, "Access denied", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, r.URL.Path[1:])
}
