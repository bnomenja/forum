package functions

import (
	"fmt"
	"net/http"
	"os"
)

func Css(w http.ResponseWriter, r *http.Request) {
	fileinfo, err := os.Stat(r.URL.Path[1:])
	if err != nil {
		RenderError(w, "Please try later", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	if fileinfo.IsDir() {
		RenderError(w, "Access denied", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, r.URL.Path[1:])
}
