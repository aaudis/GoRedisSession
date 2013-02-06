package rsess

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"math/rand"
	"net/http"
	"time"
)

/**
 * Global variables
 */
var (
	Prefix string = "sess:"
	Expire int    = 1800 // 30 minutes
)

/**
 * SessionCookie object
 */
type SessionCookie struct {
	name    string
	cookie  *http.Cookie
	values  map[string]interface{}
	clredis redis.Conn
}

/**
 * Get value of session key
 */
func (sess *SessionCookie) Get(key_name string) string {
	return fmt.Sprintf("%v", sess.values[key_name])
}

/**
 * Setting new key/updating old
 */
func (sess *SessionCookie) Set(key_name string, key_value interface{}) {
	sess.values[key_name] = key_value
	_, err := sess.clredis.Do("HSET", Prefix+sess.cookie.Value, key_name, key_value)
	if err != nil {
		log.Printf("%s", err)
	}
	expire_sess(sess) // reset expire counter
}

/**
 * Removing key
 */
func (sess *SessionCookie) Rem(key_name string) {
	delete(sess.values, key_name)
	_, err := sess.clredis.Do("HDEL", Prefix+sess.cookie.Value, key_name)
	if err != nil {
		log.Printf("%s", err)
	}
	expire_sess(sess) // reset expire counter
}

/**
 * Destroy Session/Cookie
 */
func (sess *SessionCookie) Destroy(w http.ResponseWriter) {
	sess.cookie.MaxAge = -1
	sess.values = make(map[string]interface{})
	_, err := sess.clredis.Do("DEL", Prefix+sess.cookie.Value)
	if err != nil {
		log.Printf("%s", err)
	}
	http.SetCookie(w, sess.cookie)
}

/**
 * Connect to Redis
 */
func New(session_name string, ctype, host string) (*SessionCookie, error) {
	// Creating new SessionCookie object
	sess := new(SessionCookie)
	sess.name = session_name
	sess.values = make(map[string]interface{})

	// Redis stuff
	credis, err := redis.Dial(ctype, host)
	if err != nil {
		return sess, err
	}

	// instance assign to global variable
	sess.clredis = credis
	return sess, nil
}

/**
 * Get Session - auto create Session/Cookie if not found
 */
func (sess *SessionCookie) Session(w http.ResponseWriter, r *http.Request) *SessionCookie {

	// Getting cookie
	cookie, err := r.Cookie(sess.name)
	if err != http.ErrNoCookie && err != nil {
		log.Printf("%s", err)
	}
	if cookie == nil {
		// Setting new cookie, no cookie found
		n_cookie := &http.Cookie{
			Name:    sess.name,
			Value:   get_random_value(),
			Path:    "/",
			MaxAge:  Expire,
			Expires: time.Unix(time.Now().Unix()+int64(Expire), 0),
		}
		sess.cookie = n_cookie
	} else {
		// Cookie found, getting data from Redis
		sess.cookie = cookie

		do_req, err := sess.clredis.Do("HGETALL", Prefix+sess.cookie.Value)
		if err != nil {
			log.Printf("%s", err)
		}

		v, err := redis.Values(do_req, err)
		if err != nil {
			log.Printf("%s", err)
		}

		for len(v) > 0 {
			var key, value string
			values, err := redis.Scan(v, &key, &value)
			if err != nil {
				log.Printf("%s", err)
			}
			v = values
			sess.values[key] = value
		}
	}

	// Set coookie
	http.SetCookie(w, sess.cookie)

	// return SessionCookie instance
	return sess
}

/**
 * Set Session key expire
 */
func expire_sess(sess *SessionCookie) {
	_, e := sess.clredis.Do("EXPIRE", Prefix+sess.cookie.Value, Expire)
	if e != nil {
		log.Printf("%s", e)
	}
}

/**
 * New cookie ID generator
 */
func get_random_value() string {
	rand.Seed(time.Now().UnixNano())
	c := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, 40)
	for i := 0; i < 40; i++ {
		buf[i] = c[rand.Intn(len(c)-1)]
	}
	return string(buf)
}
