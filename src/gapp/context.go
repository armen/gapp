package gapp

import (
	"github.com/armen/hdis"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/sessions"

	"bytes"
	"encoding/gob"
	"gapp/utils"
	"log"
	"net/http"
)

type Context struct {
	Session  *sessions.Session
	User     *User
	Response http.ResponseWriter
	Request  *http.Request
}

func newContext(w http.ResponseWriter, r *http.Request) (*Context, error) {

	context := &Context{Response: w, Request: r}

	// Create a session and store it in cookie so that we can recognize the user when he/she gets back
	// TODO: read session/cookie name from config
	var err error
	context.Session, err = sessionStore.Get(r, "gapp")
	if err != nil {
		log.Printf("error in getting session store: %q", err)
	}

	// Changed maximum age of the session to one month
	context.Session.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 86400 * 30,
	}

	// Generate new userid if there isn't any
	userid, ok := context.Session.Values["userid"].(string)
	if !ok {
		// It's an anonymous user
		userid = utils.GenId(16)
		context.Session.Values["userid"] = userid

	} else {
		// Let's fetch user's profile

		conn := RedisPool.Get()
		defer conn.Close()

		hc := hdis.Conn{conn}

		key := "u:" + userid + ":gob"

		if ok, _ := redis.Bool(hc.Do("HEXISTS", key)); ok {
			u, err := redis.Bytes(hc.Get(key))
			if err == nil {
				var user *User

				buffer := bytes.NewBuffer(u)
				dec := gob.NewDecoder(buffer)
				err = dec.Decode(&user)

				if err == nil {
					context.User = user
				} else {
					log.Printf("error in decoding user's (%s) profile: %q", userid, err)
				}
			} else {
				log.Printf("error in fetching user's (%s) profile from redis: %q", userid, err)
			}
		}
	}

	if context.User == nil {
		context.User = &User{Id: userid, Name: "Anonymous"}
	}

	// Saving session everytime it gets access helps to push expiry date further
	err = context.Session.Save(r, w)

	return context, err
}
