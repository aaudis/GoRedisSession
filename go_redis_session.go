package rsess

import (
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
	cl_redis  redis.Conn
	Prefix    string = "sess:"
	CookieAge int    = 1800 // 30 minutes
)

/**
 * SessionCookie object
 */
type SessionCookie struct {
	name   string
	cookie *http.Cookie
	values map[string]interface{}
}

/**
 * Setting new key/updating old
 */
func (sess *SessionCookie) Set(key_name string, key_value interface{}) {
	sess.values[key_name] = key_value
	_, err := cl_redis.Do("HSET", Prefix+sess.cookie.Value, key_name, key_value)
	if err != nil {
		log.Printf("%s", err)
	}
}

/**
 * Removing key
 */
func (sess *SessionCookie) Rem(key_name string) {
	delete(sess.values, key_name)
	_, err := cl_redis.Do("HDEL", Prefix+sess.cookie.Value, key_name)
	if err != nil {
		log.Printf("%s", err)
	}
}

/**
 * Destroy Session/Cookie
 */
func (sess *SessionCookie) Destroy(w http.ResponseWriter) {
	sess.cookie.MaxAge = -1
	sess.values = make(map[string]interface{})
	_, err := cl_redis.Do("DEL", Prefix+sess.cookie.Value)
	if err != nil {
		log.Printf("%s", err)
	}
	http.SetCookie(w, sess.cookie)
}

/**
 * Connect to Redis
 */
func Connect(ctype, host string) error {
	credis, err := redis.Dial(ctype, host)
	if err != nil {
		return err
	}

	// instance assign to global variable
	cl_redis = credis
	return nil
}

/**
 * Get Session - auto create Session/Cookie if not found
 */
func Get(w http.ResponseWriter, r *http.Request, session_name string) *SessionCookie {
	// Setting init params
	sess := new(SessionCookie)
	sess.name = session_name
	sess.values = make(map[string]interface{})

	// Getting cookie
	cookie, err := r.Cookie(session_name)
	if err != nil {
		log.Printf("%s", err)
	}
	if cookie == nil {
		// Setting new cookie, no cookie found
		n_cookie := &http.Cookie{
			Name:    session_name,
			Value:   get_random_value(),
			Path:    "/",
			MaxAge:  CookieAge,
			Expires: time.Unix(time.Now().Unix()+int64(CookieAge), 0),
		}
		sess.cookie = n_cookie
		http.SetCookie(w, n_cookie)
	} else {
		// Cookie found, getting data from Redis
		sess.cookie = cookie

		do_req, err := cl_redis.Do("HGETALL", Prefix+sess.cookie.Value)
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

		// Set EXPIRE age in Redis
		_, e := cl_redis.Do("EXPIRE", Prefix+sess.cookie.Value, CookieAge)
		if e != nil {
			log.Printf("%s", e)
		}
	}

	// return SessionCookie instance
	return sess
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
