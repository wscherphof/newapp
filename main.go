package main

import (
	essix "github.com/wscherphof/essix/server"
	"github.com/wscherphof/newapp/messages"
)

func init() {
	messages.Init()
}

func main() {
	essix.Run()
}
