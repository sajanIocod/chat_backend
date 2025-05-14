package utils

import (
	"log"
	"os"

	"github.com/pusher/pusher-http-go/v5"
)

var PusherClient pusher.Client

func InitPusher() {
	PusherClient = pusher.Client{
		AppID:   os.Getenv("PUSHER_APP_ID"),
		Key:     os.Getenv("PUSHER_KEY"),
		Secret:  os.Getenv("PUSHER_SECRET"),
		Cluster: os.Getenv("PUSHER_CLUSTER"),
		Secure:  true,
	}

	log.Println("[INFO] Pusher initialized successfully")
}
