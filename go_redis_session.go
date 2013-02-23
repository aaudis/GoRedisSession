package rsess

import (
	rsess_redis_connector "./redis"
	"fmt"
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
	Expire  int64  = 1800 // 30 minutes
	clredis rsess_redis_connector.Client
)

/*
	SessionConnect object
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
	values map[string]string
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
func (sess *SessionCookie) Set(key_name, key_value string) {
	sess.values[key_name] = key_value
	err := clredis.Hset(Prefix+sess.cookie.Value, key_name, []byte(key_value))
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
	_, err := clredis.Hdel(Prefix+sess.cookie.Value, key_name)
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
	sess.cookie.Expires = time.Now().AddDate(0, 0, -1)
	sess.values = make(map[string]string)
	_, err := clredis.Del(Prefix + sess.cookie.Value)
	if err != nil {
		log.Printf("%s", err)
	}
	http.SetCookie(w, sess.cookie)
}

/*
	Connect to Redis and returning instance of SessionConnect
*/
func New(session_name string, database int, host string, port int) (*SessionConnect, error) {
	// session ID name
	temp_connection := new(SessionConnect)
	temp_connection.session_id = session_name

	// connecting to redis
	tmp_redis := rsess_redis_connector.DefaultSpec().Db(database).Host(host).Port(port)
	tmp_client, e := rsess_redis_connector.NewSynchClientWithSpec(tmp_redis)
	if e != nil {
		log.Printf("%s", e)
		return nil, e
	}
	clredis = tmp_client

	// assign Redis connection
	return temp_connection, nil
}

/*
	Get Session - auto create Session and Cookie if not found
*/
func (conn *SessionConnect) Session(w http.ResponseWriter, r *http.Request) *SessionCookie {
	// New cookie object
	t_sess := new(SessionCookie)
	t_sess.name = conn.session_id
	t_sess.values = make(map[string]string)

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
			MaxAge:  int(Expire),
			Expires: time.Unix(time.Now().Unix()+Expire, 0),
		}
		t_sess.cookie = n_cookie
	} else {
		// Cookie found, getting data from Redis
		t_sess.cookie = cookie

		req, err := clredis.Hgetall(Prefix + t_sess.cookie.Value)
		if err != nil {
			log.Printf("%s", err)
		}

		iskey := true
		keyname := ""
		for _, item := range req {
			if iskey {
				t_sess.values[string(item)] = ""
				keyname = string(item)
				iskey = false
			} else {
				t_sess.values[keyname] = string(item)
				iskey = true
			}
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
	_, e := clredis.Expire(Prefix+sess.cookie.Value, Expire)
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
