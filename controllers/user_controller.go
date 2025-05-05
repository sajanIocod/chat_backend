package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sajanIocod/chat_backend/models"
	"github.com/sajanIocod/chat_backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetUsers(c *gin.Context) {
	search := c.Query("search")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	userID := c.GetString("userID")

	// Convert page & limit to integers
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	skip := (page - 1) * limit

	// Build MongoDB filter
	filter := bson.M{}
	if search != "" {
		filter["username"] = bson.M{
			"$regex":   search,
			"$options": "i",
		}
	}
	filter["_id"] = bson.M{"$ne": utils.ObjectIDFromHex(userID)}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MongoDB find with skip and limit
	options := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit))
	cursor, err := utils.DB.Collection("users").Find(ctx, filter, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching users"})
		return
	}

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing users"})
		return
	}

	// Format response
	var userList []gin.H
	for _, u := range users {
		// Count unread messages from this user
		unreadCount, err := utils.DB.Collection("messages").CountDocuments(ctx, bson.M{
			"senderId":   u.ID,
			"receiverId": utils.ObjectIDFromHex(userID),
			"seen":       false,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.Response{
				ResponseCode: http.StatusInternalServerError,
				Message:      "Error counting unread messages",
				Data:         nil,
			})
			return

		}

		userList = append(userList, gin.H{
			"id":          u.ID.Hex(),
			"username":    u.Username,
			"email":       u.Email,
			"unreadCount": unreadCount,
		})
	}

	// Count unread messages from a user

	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Users fetched successfully",
		Data:         userList,
	})
}
