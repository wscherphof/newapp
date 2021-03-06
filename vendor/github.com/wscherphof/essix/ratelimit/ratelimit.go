/*
Package ratelimit manages rate limits for urls.

It can be used instead of captchas, since breaking captchas is a commercial
service, yielding a captcha to be nothing more than a rate limit.

The rate limited requests for each client (ip address) are stored in the
database, and subsequent requests from the same client are denied if they come
within the set time window.

Entries of non-returning clients are cleared from the database regularly.
*/
package ratelimit

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/wscherphof/entity"
	"github.com/wscherphof/env"
	"github.com/wscherphof/essix/template"
	"github.com/wscherphof/secure"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	ErrTooManyRequests = errors.New("ErrTooManyRequests")
	ErrInvalidRequest  = errors.New("ErrInvalidRequest")
	defaultLimit       = env.GetInt("RATELIMIT", 60)
)

const (
	clearInterval = time.Hour
)

type path string

type requests map[path]time.Time

type client struct {
	*entity.Base
	Clear    int64
	Requests requests
}

func init() {
	entity.Register(&client{}).Index("Clear")
	secure.RegisterRequestTokenData(token{})
	go func() {
		for {
			time.Sleep(clearInterval)
			limit := time.Now().Unix()
			index := entity.Index(&client{}, "Clear")
			selection := index.Between(nil, true, limit, true)
			if deleted, err := selection.Delete(); err != nil {
				log.Printf("WARNING: rate limit clearing failed: %v", err)
			} else {
				log.Printf("INFO: %v rate limit records cleared", deleted)
			}
		}
	}()
}

type token struct {
	IP        string
	Path      path
	Timestamp time.Time
}

func ip(r *http.Request) string {
	return strings.Split(r.RemoteAddr, ":")[0]
}

/*
NewToken returns an encrypted rate limit token to include in a rate limited
request.
*/
func NewToken(r *http.Request, differentPath ...string) (string, error) {
	t := &token{
		IP:        ip(r),
		Timestamp: time.Now(),
	}
	if len(differentPath) == 1 {
		t.Path = path(differentPath[0])
	} else {
		t.Path = path(r.URL.Path)
	}
	return secure.NewRequestToken(t)
}

func getClient(ip string) (c *client) {
	c = &client{Base: &entity.Base{ID: ip}}
	if err, empty := c.Read(c); err != nil {
		if empty {
			c.Requests = make(requests, 20)
		} else {
			log.Printf("WARNING: error reading from ratelimit table: %v", err)
		}
	}
	return
}

/*
ratelimit.Handle returns a httprouter.Handle that denies a request if it's
repeated from the same client within the given number of seconds, or handles it,
and resets the timw window.

Default limit is read from the RATELIMIT environment variable. The default value
for RATELIMIT is 60 seconds.
*/
func Handle(handle httprouter.Handle, opt_seconds ...int) httprouter.Handle {
	seconds := defaultLimit
	if len(opt_seconds) == 1 {
		seconds = opt_seconds[0]
	}
	window := time.Duration(seconds) * time.Second
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		t, ip, p := new(token), ip(r), path(r.URL.Path)
		if formToken := r.FormValue("_ratelimit"); formToken == "" {
			template.Error(w, r, ErrInvalidRequest, true)
			log.Printf("SUSPICIOUS: rate limit token missing %v %v", ip, p)
		} else if e := secure.RequestToken(formToken).Read(t); e != nil {
			template.Error(w, r, ErrInvalidRequest, true)
			log.Printf("SUSPICIOUS: rate limit token unreadable %v %v", ip, p)
		} else if t.IP != ip {
			template.Error(w, r, ErrInvalidRequest, true)
			log.Printf("SUSPICIOUS: rate limit token invalid address: %v, expected %v %v", t.IP, ip, p)
		} else if t.Path != p {
			template.Error(w, r, ErrInvalidRequest, true)
			log.Printf("SUSPICIOUS: rate limit token invalid path: %v, token path %v, expected %v", ip, t.Path, p)
		} else if c := getClient(ip); c.Requests[p].After(t.Timestamp) {
			template.Error(w, r, ErrInvalidRequest, true)
			log.Printf("SUSPICIOUS: rate limit token reuse: %v %v, token %v, previous request %v", ip, p, t.Timestamp, c.Requests[p])
		} else if c.Requests[p].After(time.Now().Add(-window)) {
			template.ErrorTail(w, r, ErrTooManyRequests, true, "ratelimit", "TooManyRequests", map[string]interface{}{
				"window": window,
			})
		} else {
			c.Requests[p] = time.Now()
			if clear := time.Now().Add(window).Unix(); clear > c.Clear {
				c.Clear = clear
			}
			if e := c.Update(c); e != nil {
				log.Printf("WARNING: error updating ratelimit record: %v", e)
			}
			handle(w, r, ps)
		}
		return
	}
}
