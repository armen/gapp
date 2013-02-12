package gapp

import (
	"github.com/gorilla/sessions"

	"gapp/utils"
	"net/http"
)

type Context struct {
	Session *sessions.Session
	User    *User
}

func newContext(w http.ResponseWriter, r *http.Request) (*Context, error) {

	context := &Context{}

	// Create a session and store it in cookie so that we can recognize the user when he/she gets back
	// TODO: read session/cookie name from config
	var err error
	context.Session, err = sessionStore.Get(r, "gapp")
	if err != nil {
		return nil, err
	}

	// Changed maximum age of the session to one month
	context.Session.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 86400 * 30,
	}

	// Generate new userid if there isn't any
	userid, ok := context.Session.Values["userid"]
	if !ok {
		userid = utils.GenId(16)
		context.Session.Values["userid"] = userid
	}

	// Saving session everytime it gets access helps to push expiry date further
	err = context.Session.Save(r, w)

	return context, err
}
