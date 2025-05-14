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
)

func GetChatList(c *gin.Context) {
	userID := c.GetString("userID")
	currentUserID := utils.ObjectIDFromHex(userID)

	// Get pagination and search parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	search := c.Query("search")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	skip := (page - 1) * limit

	log.Printf("[INFO] Fetching chat list for user: %s (page: %d, limit: %d, search: %s)",
		userID, page, limit, search)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build search match stage
	searchMatch := bson.M{}
	if search != "" {
		searchMatch = bson.M{
			"$or": []bson.M{
				{"user.username": bson.M{"$regex": search, "$options": "i"}},
				{"lastMessage.content": bson.M{"$regex": search, "$options": "i"}},
			},
		}
	}

	// First get total count
	countPipeline := []bson.M{
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"senderID": currentUserID},
					{"receiverID": currentUserID},
				},
			},
		},
		{
			"$sort": bson.M{"createdAt": -1},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"$cond": []interface{}{
						bson.M{"$eq": []interface{}{"$senderID", currentUserID}},
						"$receiverID",
						"$senderID",
					},
				},
				"lastMessage": bson.M{"$first": "$$ROOT"},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "user",
			},
		},
		{
			"$unwind": "$user",
		},
	}

	// Add search match if search parameter is provided
	if search != "" {
		countPipeline = append(countPipeline, bson.M{"$match": searchMatch})
	}

	// Add count stage
	countPipeline = append(countPipeline, bson.M{"$count": "total"})

	var countResult []bson.M
	cursor, err := utils.DB.Collection("messages").Aggregate(ctx, countPipeline)
	if err != nil {
		log.Printf("[ERROR] Failed to count chats: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error counting chats",
			Data:         nil,
		})
		return
	}
	if err := cursor.All(ctx, &countResult); err != nil {
		log.Printf("[ERROR] Failed to decode count result: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error processing chat count",
			Data:         nil,
		})
		return
	}

	totalChats := int64(0)
	if len(countResult) > 0 {
		totalChats = countResult[0]["total"].(int64)
	}

	// Main pipeline with search and pagination
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"senderID": currentUserID},
					{"receiverID": currentUserID},
				},
			},
		},
		{
			"$sort": bson.M{"createdAt": -1},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"$cond": []interface{}{
						bson.M{"$eq": []interface{}{"$senderID", currentUserID}},
						"$receiverID",
						"$senderID",
					},
				},
				"lastMessage": bson.M{"$first": "$$ROOT"},
				"unreadCount": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{
								"$and": []bson.M{
									{"$eq": []interface{}{"$receiverID", currentUserID}},
									{"$eq": []interface{}{"$seen", false}},
								},
							},
							1,
							0,
						},
					},
				},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "user",
			},
		},
		{
			"$unwind": "$user",
		},
	}

	// Add search match if search parameter is provided
	if search != "" {
		pipeline = append(pipeline, bson.M{"$match": searchMatch})
	}

	// Add projection, skip and limit stages
	pipeline = append(pipeline,
		bson.M{
			"$project": bson.M{
				"userId":          "$_id",
				"username":        "$user.username",
				"email":           "$user.email",
				"lastMessage":     "$lastMessage.content",
				"lastMessageTime": "$lastMessage.createdAt",
				"unreadCount":     1,
			},
		},
		bson.M{"$skip": skip},
		bson.M{"$limit": limit},
	)

	cursor, err = utils.DB.Collection("messages").Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch chat list: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error fetching chat list",
			Data:         nil,
		})
		return
	}
	defer cursor.Close(ctx)

	var chatList []gin.H
	if err := cursor.All(ctx, &chatList); err != nil {
		log.Printf("[ERROR] Failed to decode chat list: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error processing chat list",
			Data:         nil,
		})
		return
	}

	totalPages := (totalChats + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Chat list fetched successfully",
		Data: gin.H{
			"chats": chatList,
			"pagination": gin.H{
				"currentPage": page,
				"limit":       limit,
				"totalItems":  totalChats,
				"totalPages":  totalPages,
			},
		},
	})
}
