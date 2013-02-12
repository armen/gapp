package gapp

import (
	"github.com/gorilla/mux"

	"fmt"
	"net/http"
	"path"
	"strings"
)

func pageHandler(c *Context) error {
	vars := mux.Vars(c.Request)
	page := vars["page"]

	switch page {
	case "about", "privacy", "feedback":

		data := map[string]interface{}{
			"BUILD":    BuildId,
			"page":     map[string]bool{page: true}, // Select the page in the top navbar
			"title":    strings.Title(page),
			"keywords": page,
		}

		err := Templates.ExecuteTemplate(c.Response, path.Join(page+".html"), data)
		if err != nil {
			return err
		}
	default:
		err := fmt.Errorf("Page %q could not be found!", page)
		return &HandlerError{Err: err, Message: err.Error(), Code: http.StatusNotFound}
	}

	return nil
}
