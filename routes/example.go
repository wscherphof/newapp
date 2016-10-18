package routes

import (
	"github.com/wscherphof/essix/router"
	"github.com/wscherphof/newapp/routes/example"
	"github.com/wscherphof/essix/secure"
)

func init() {
	router.GET("/profile", secure.Handle(example.ProfileForm))
	router.PUT("/profile", secure.Handle(account.Profile))
}
