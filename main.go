package main

import (
	essix "github.com/wscherphof/essix/server"
	"github.com/wscherphof/newapp/messages"
	"github.com/wscherphof/newapp/routes"
)

func init() {
	messages.Init()
	routes.Init()
}

func main() {
	essix.Run()
}
