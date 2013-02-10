package rsess

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"math/rand"
	"net/http"
	"time"
)

/* 
	Global variables
*/
var (
	Prefix  string = "sess:"
	Expire  int    = 1800 // 30 minutes
	clredis redis.Conn
)

/*
	Connection object
*/
type SessionConnect struct {
	session_id string
}

/*
	SessionCookie object
*/
type SessionCookie struct {
	name   string
	cookie *http.Cookie
	values map[string]interface{}
}

/*
	Get value of session key
*/
func (sess *SessionCookie) Get(key_name string) string {
	return fmt.Sprintf("%v", sess.values[key_name])
}

/*
	Setting new key, updating old
*/
func (sess *SessionCookie) Set(key_name string, key_value interface{}) {
	sess.values[key_name] = key_value
	_, err := clredis.Do("HSET", Prefix+sess.cookie.Value, key_name, key_value)
	if err != nil {
		log.Printf("%s", err)
	}
	expire_sess(sess) // reset expire counter
}

/*
	Removing key
*/
func (sess *SessionCookie) Rem(key_name string) {
	delete(sess.values, key_name)
	_, err := clredis.Do("HDEL", Prefix+sess.cookie.Value, key_name)
	if err != nil {
		log.Printf("%s", err)
	}
	expire_sess(sess) // reset expire counter
}

/*
	Destroy Session/Cookie
*/
func (sess *SessionCookie) Destroy(w http.ResponseWriter) {
	sess.cookie.MaxAge = -1
	sess.values = make(map[string]interface{})
	_, err := clredis.Do("DEL", Prefix+sess.cookie.Value)
	if err != nil {
		log.Printf("%s", err)
	}
	http.SetCookie(w, sess.cookie)
}

/*
	Set Redis database
*/
func (sess *SessionCookie) Database(db int) {
	_, err := clredis.Do("SELECT", db)
	if err != nil {
		log.Printf("%s", err)
	}
}

/*
	Connect to Redis and returning instance of SessionCookie
*/
func New(session_name string, database int, ctype, host string) (*SessionConnect, error) {
	// session ID name
	temp_connection := new(SessionConnect)
	temp_connection.session_id = session_name

	// connecting to redis
	credis, err := redis.Dial(ctype, host)
	if err != nil {
		return nil, err
	}

	// Select Redis DB
	def := 0
	if database > 0 {
		def = database
	}
	_, e := credis.Do("SELECT", def)
	if e != nil {
		log.Printf("%s", e)
	}

	// assign Redis connection
	clredis = credis
	return temp_connection, nil
}

/*
	Get Session - auto create Session/Cookie if not found
*/
func (conn *SessionConnect) Session(w http.ResponseWriter, r *http.Request) *SessionCookie {
	// New cookie object
	t_sess := new(SessionCookie)
	t_sess.name = conn.session_id
	t_sess.values = make(map[string]interface{})

	// Getting cookie
	cookie, err := r.Cookie(t_sess.name)
	if err != http.ErrNoCookie && err != nil {
		log.Printf("%s", err)
	}
	if cookie == nil {
		// Setting new cookie, no cookie found
		n_cookie := &http.Cookie{
			Name:    t_sess.name,
			Value:   get_random_value(),
			Path:    "/",
			MaxAge:  Expire,
			Expires: time.Unix(time.Now().Unix()+int64(Expire), 0),
		}
		t_sess.cookie = n_cookie
	} else {
		// Cookie found, getting data from Redis
		t_sess.cookie = cookie

		do_req, err := clredis.Do("HGETALL", Prefix+t_sess.cookie.Value)
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
			t_sess.values[key] = value
		}

		// reset expiration
		expire_sess(t_sess)
	}

	// Set coookie
	http.SetCookie(w, t_sess.cookie)

	// return SessionCookie instance
	return t_sess
}

/*
	Set Session key expire
*/
func expire_sess(sess *SessionCookie) {
	_, e := clredis.Do("EXPIRE", Prefix+sess.cookie.Value, Expire)
	if e != nil {
		log.Printf("%s", e)
	}
}

/*
	New cookie ID generator
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
