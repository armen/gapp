package sessions

import (
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"encoding/base32"
	"net/http"
	"strings"
)

// NewRedisStore returns a new RedisStore
func NewRedisStore(pool *redis.Pool, keyPairs ...[]byte) *RedisStore {

	return &RedisStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		redisPool: pool,
	}
}

// RedisStore stores sessions in redis.
type RedisStore struct {
	Codecs    []securecookie.Codec
	Options   *sessions.Options
	redisPool *redis.Pool
}

// Get returns a session for the given name after adding it to the registry.
func (s *RedisStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
func (s *RedisStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	session.Options = &(*s.Options)
	session.IsNew = true
	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.load(session)
			if err == nil {
				session.IsNew = false
			}
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *RedisStore) Save(r *http.Request, w http.ResponseWriter,
	session *sessions.Session) error {
	if session.ID == "" {
		session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
	}
	if err := s.save(session); err != nil {
		return err
	}
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

// save writes encoded session.Values into redis.
func (s *RedisStore) save(session *sessions.Session) error {

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}

	c := s.redisPool.Get()
	defer c.Close()

	_, err = c.Do("SET", "s:"+session.ID, encoded)
	if err != nil {
		return err
	}

	_, err = c.Do("EXPIRE", "s:"+session.ID, session.Options.MaxAge)

	return err
}

// load reads from redis and decodes its content into session.Values.
func (s *RedisStore) load(session *sessions.Session) error {

	c := s.redisPool.Get()
	defer c.Close()

	encoded, err := redis.String(c.Do("GET", "s:"+session.ID))
	if err != nil {
		return err
	}

	return securecookie.DecodeMulti(session.Name(), encoded, &session.Values, s.Codecs...)
}
