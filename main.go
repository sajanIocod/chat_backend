package main

import (
	"github.com/sajanIocod/chat_backend/routes"
	"github.com/sajanIocod/chat_backend/utils"
)

func main() {
	utils.ConnectDB()
	utils.InitPusher()
	r := routes.SetupRouter()
	r.Run(":8080")
}
