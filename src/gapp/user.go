package gapp

import (
	"github.com/armen/hdis"
	"github.com/garyburd/redigo/redis"
	"github.com/nfnt/resize"

	"bytes"
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"
)

type User struct {
	Id            string
	SignedIn      bool
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

func NewUser(id string) *User {
	return &User{Id: id}
}

var (
	previousPicture string
)

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

	err := user.PreLogin(c)
	if err != nil {
		return err
	}

	// Signin the user
	user.SignedIn = true

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

	err = user.PostLogin(c)
	return err
}

func (user *User) PreLogin(c *Context) error {

	if user.Id != "" {
		conn := RedisPool.Get()
		defer conn.Close()

		hc := hdis.Conn{conn}
		previousPicture, _ = redis.String(hc.Get("u:" + user.Id + ":picture"))
	}

	return nil
}

func (user *User) PostLogin(c *Context) error {

	if previousPicture != user.Picture {

		tr := &http.Transport{
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			DisableCompression: true,
		}
		client := &http.Client{Transport: tr}

		resp, err := client.Get(user.Picture)
		if err != nil {
			// Something is wrong with the picture, lets check it next time
			user.Picture = ""
			log.Println(err)
			return nil
		}
		defer resp.Body.Close()

		// Decode user's image
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			// Something is wrong with the picture, lets check it next time
			user.Picture = ""
			log.Println(err)
			return nil
		}

		thumbPath := path.Join(AppRoot, "static", "users", user.Id)
		err = os.MkdirAll(thumbPath, 0777)
		if err != nil {
			user.Picture = ""
			log.Println(err)
			return nil
		}

		avatars := []uint{32, 92}

		for _, size := range avatars {

			resizedImg := resize.Resize(size+1, size+1, img, resize.Lanczos3)
			thumb, err := os.Create(path.Join(thumbPath, fmt.Sprintf("avatar-%dx%d.jpg", size, size)))
			if err != nil {
				user.Picture = ""
				os.RemoveAll(thumbPath)
				log.Println(err)
				return nil
			}
			defer thumb.Close()

			// remove left and top's, 1px border
			rect := image.Rect(0, 0, int(size), int(size))
			final := image.NewRGBA(rect)
			draw.Draw(final, rect, resizedImg, image.Point{1, 1}, draw.Src)

			err = jpeg.Encode(thumb, final, nil)
			if err != nil {
				user.Picture = ""
				os.RemoveAll(thumbPath)
				log.Println(err)
				return nil
			}
		}
	}

	return nil
}

func (user *User) Logout(c *Context) error {

	err := user.PreLogout(c)
	if err != nil {
		return err
	}

	// MaxAge < 0 deletes the session
	c.Session.Options.MaxAge = -1
	c.Session.Save(c.Request, c.Response)

	err = user.PostLogout(c)
	return err
}

func (user *User) PreLogout(c *Context) error {
	return nil
}

func (user *User) PostLogout(c *Context) error {
	return nil
}

func (user *User) Profile(attribute string) string {
	conn := RedisPool.Get()
	defer conn.Close()

	hc := hdis.Conn{conn}

	attr, _ := redis.String(hc.Get("u:" + user.Id + ":" + strings.ToLower(attribute)))

	return attr
}
