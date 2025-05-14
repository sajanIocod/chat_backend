package controllers

import (
	"context"
	"log"
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

	log.Printf("[INFO] Fetching users - Search: '%s', Page: %s, Limit: %s, UserID: %s",
		search, pageStr, limitStr, userID)

	// Convert page & limit to integers
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	skip := (page - 1) * limit

	// Build MongoDB filter
	filter := bson.M{
		"_id": bson.M{"$ne": utils.ObjectIDFromHex(userID)}, // Exclude current user
	}

	// Add search filter only if search parameter is provided
	if search != "" {
		filter["username"] = bson.M{
			"$regex":   search,
			"$options": "i",
		}
		log.Printf("[INFO] Applying search filter for username: %s", search)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// MongoDB find with skip and limit
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"username": 1}) // Sort by username alphabetically

	cursor, err := utils.DB.Collection("users").Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch users: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error fetching users",
			Data:         nil,
		})
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		log.Printf("[ERROR] Failed to parse users: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error parsing users",
			Data:         nil,
		})
		return
	}

	// Get total count for pagination
	totalCount, err := utils.DB.Collection("users").CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("[ERROR] Failed to count total users: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error counting total users",
			Data:         nil,
		})
		return
	}

	// Format response with unread message counts
	userList := make([]gin.H, 0, len(users))
	for _, u := range users {
		// Count unread messages from this user
		unreadCount, err := utils.DB.Collection("messages").CountDocuments(ctx, bson.M{
			"senderID":   u.ID,
			"receiverID": utils.ObjectIDFromHex(userID),
			"seen":       false,
		})
		if err != nil {
			log.Printf("[ERROR] Failed to count unread messages from user %s: %v", u.ID.Hex(), err)
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

	// Return response with pagination info
	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Users fetched successfully",
		Data: gin.H{
			"users": userList,
			"pagination": gin.H{
				"currentPage": page,
				"limit":       limit,
				"totalItems":  totalCount,
				"totalPages":  (totalCount + int64(limit) - 1) / int64(limit),
			},
		},
	})
}
