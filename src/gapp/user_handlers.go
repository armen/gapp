package gapp

import (
	"code.google.com/p/goauth2/oauth"
	"github.com/gorilla/sessions"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type User struct {
	ID            string
	OauthID       string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email",bool`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Link          string `json:"link"`
	Gender        string `json:"gender"`
	Locale        string `json:"locale"`
	HostDomain    string `json:"hd"`
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

func signinHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) error {

	data := map[string]interface{}{
		"BUILD":    BuildId,
		"page":     map[string]bool{"signin": true}, // Select signin in the top navbar
		"title":    "Signin & Signup",
		"keywords": "signin, signup, loging, register",
	}

	err := Templates.ExecuteTemplate(w, "signin.html", data)
	if err != nil {
		return err
	}

	return nil
}

func googleSigninHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) error {

	initOauthCfg()

	// Replace %s with hostname
	oauthCfg.RedirectURL = fmt.Sprintf(redirectURL, strings.TrimSpace(Hostname))

	// Get the Google URL which shows the Authentication page to the user
	url := oauthCfg.AuthCodeURL("")

	// Redirect user to that page
	http.Redirect(w, r, url, http.StatusFound)

	return nil
}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) error {

	userid := s.Values["userid"].(string)

	// Get the code from the response
	code := r.FormValue("code")

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
	defer r.Body.Close()

	c := RedisPool.Get()
	defer c.Close()

	var user User
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return err
	}

	user.ID = userid
	fmt.Fprintf(w, "%#v", user)

	return nil
}
