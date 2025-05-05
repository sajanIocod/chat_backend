package main

import (
	"github.com/sajanIocod/chat_backend/routes"
	"github.com/sajanIocod/chat_backend/utils"
)

func main() {
	utils.ConnectDB()
	r := routes.SetupRouter()
	r.Run(":8080")
}
