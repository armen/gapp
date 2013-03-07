package gapp

import (
	"code.google.com/p/goauth2/oauth"
	"github.com/armen/hdis"
	"github.com/garyburd/redigo/redis"

	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
)

type User struct {
	Id            string
	OauthId       string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email",bool`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Link          string `json:"link"`
	Gender        string `json:"gender"`
	Locale        string `json:"locale"`
	HostDomain    string `json:"hd"`
	Picture       string `json:"picture"`
	Birthday      string `json:"birthday"`
}

const (
	redirectURL    = "http://%s/google-callback"
	profileInfoURL = "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"
)

var (
	oauthCfg *oauth.Config
)

func initOauthCfg() {

	clientId, err := Config.GetString("google", "client-id")
	if err != nil {
		log.Println(err)
	}

	clientSecret, err := Config.GetString("google", "client-secret")
	if err != nil {
		log.Println(err)
	}

	oauthCfg = &oauth.Config{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		Scope:        "https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
	}
}

func signinHandler(c *Context) error {

	data := map[string]interface{}{
		"BUILD":    BuildId,
		"page":     map[string]bool{"signin": true}, // Select signin in the top navbar
		"title":    "Signin & Signup",
		"keywords": "signin, signup, loging, register",
	}

	// TODO: check if user is already logged on

	err := Templates.ExecuteTemplate(c.Response, "signin.html", data)
	if err != nil {
		return err
	}

	return nil
}

func googleSigninHandler(c *Context) error {

	initOauthCfg()

	// Replace %s with hostname
	oauthCfg.RedirectURL = fmt.Sprintf(redirectURL, strings.TrimSpace(Hostname))

	// Get the Google URL which shows the Authentication page to the user
	url := oauthCfg.AuthCodeURL("")

	// Redirect user to that page
	http.Redirect(c.Response, c.Request, url, http.StatusFound)

	return nil
}

func googleCallbackHandler(c *Context) error {

	// Get the code from the response
	code := c.Request.FormValue("code")

	t := &oauth.Transport{oauth.Config: oauthCfg}

	// Exchange the received code for a token
	_, err := t.Exchange(code)
	if err != nil {
		return err
	}

	// Now get user's data based on the Transport which has the token
	resp, err := t.Client().Get(profileInfoURL)
	if err != nil {
		return err
	}
	defer c.Request.Body.Close()

	// For development purpose resp.Body is logged but in the future
	// it can be used in the decoder directly
	//
	//      err = json.NewDecoder(resp.Body).Decode(&user)
	//
	content, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(content))

	var user *User
	err = json.Unmarshal(content, &user)
	if err != nil {
		return err
	}

	err = user.Login(c)
	if err != nil {
		return err
	}

	// Put the user in the context
	c.User = user

	return nil
}

func (user *User) Login(c *Context) error {

	conn := RedisPool.Get()
	defer conn.Close()

	hc := hdis.Conn{conn}

	user.Id, _ = redis.String(conn.Do("ZSCORE", "users", user.Email))

	if user.Id == "" {

		// Yay another newcomer!
		nextuserid, err := conn.Do("INCR", "next-user-id")
		if err != nil {
			return err
		}

		// Pad userid with 0
		user.Id = fmt.Sprintf("%03d", nextuserid.(int64))

		_, err = conn.Do("ZADD", "users", user.Id, user.Email)
		if err != nil {
			return err
		}

	} else {
		// Pad user.Id with 0
		user.Id = fmt.Sprintf("%03s", user.Id)
	}

	// Replace c.User.Id with new userid
	c.Session.Values["userid"] = user.Id
	err := c.Session.Save(c.Request, c.Response)
	if err != nil {
		return err
	}

	// Encode to gob
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err = enc.Encode(user)
	if err != nil {
		return err
	}

	// Every time set the user's info, actually it'll update the profile
	key := "u:" + user.Id + ":gob"
	_, err = hc.Set(key, buffer.Bytes())
	if err != nil {
		return err
	}

	keys := []string{"Name", "GivenName", "FamilyName", "Email", "Gender", "Picture", "Birthday"}
	for _, key := range keys {
		rediskey := "u:" + user.Id + ":" + strings.ToLower(key)
		_, err = hc.Set(rediskey, reflect.ValueOf(user).Elem().FieldByName(key).String())
		if err != nil {
			return err
		}
	}

	http.Redirect(c.Response, c.Request, "/", http.StatusSeeOther)

	return nil
}
