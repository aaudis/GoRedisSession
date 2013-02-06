# Redis Sessions in Go (golang)
Simple session storage in Redis

## Install
```bash
$ go get github.com/aaudis/GedisSession
```

## Documentation
http://godoc.org/github.com/aaudis/GoRedisSession

## Simple usage
```go
package main

import (
	"fmt"
	"github.com/aaudis/GoRedisSession"
	"log"
	"net/http"
)

var (
	redis_session *rsess.SessionCookie
)

func main() {
	// Configurable parameters
	rsess.Prefix = "sess:" // session prefix (in Redis)
	rsess.Expire = 1800    // 30 minute session expiration

	// Connecting to Redis and creating storage instance
	temp_sess, err := rsess.New("sid", "tcp", "127.0.0.1:6379")
	if err != nil {
		log.Printf("%s", err)
	}
	redis_session = temp_sess

	http.HandleFunc("/", Root)
	http.HandleFunc("/get", Get)
	http.HandleFunc("/set", Set)
	http.ListenAndServe(":8888", nil)
}

func Root(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprintf(w, `
  		Redis session storage example:<br><br>
  		<a href="/set">Store key in session</a><br>
  		<a href="/get">Get key value from session</a>
  	`)
}

func Set(w http.ResponseWriter, r *http.Request) {
	s := redis_session.Session(w, r)
	s.Set("UserID", 1000)
	fmt.Fprintf(w, "Setting session variable done!")
}

func Get(w http.ResponseWriter, r *http.Request) {
	s := redis_session.Session(w, r)
	fmt.Fprintf(w, "Value %s", s.Get("UserID"))
}
```