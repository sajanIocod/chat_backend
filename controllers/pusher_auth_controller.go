package controllers

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/sajanIocod/chat_backend/utils"
)

func PusherAuth(c *gin.Context) {
	socketID := c.PostForm("socket_id")
	channel := c.PostForm("channel_name")

	// Get user ID from JWT token (assuming you have middleware setting this)
	userID := c.GetString("userID")
	// Create request payload
	payload := map[string]interface{}{
		"socket_id":    socketID,
		"channel_name": channel,
		"user_id":      userID,
	}

	// Marshal to []byte
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "Failed to marshal payload: " + err.Error(),
		})
		return
	}

	response, err := utils.PusherClient.AuthorizePrivateChannel(payloadBytes)
	if err != nil {
		c.JSON(403, gin.H{
			"error": "Failed to authenticate Pusher channel: " + err.Error(),
		})
		return
	}

	c.JSON(200, response)
}
