package gapp

import (
	"code.google.com/p/goauth2/oauth"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

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
		"user":     &c.User,
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

func signoutHandler(c *Context) error {

	err := c.User.Logout(c)
	if err != nil {
		return err
	}

	http.Redirect(c.Response, c.Request, "/", http.StatusFound)

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

	// Update userid in session
	c.Session.Values["userid"] = c.User.Id
	err = c.Session.Save(c.Request, c.Response)
	if err != nil {
		return err
	}

	http.Redirect(c.Response, c.Request, "/", http.StatusFound)

	return nil
}
