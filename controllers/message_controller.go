package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/sajanIocod/chat_backend/models"
	"github.com/sajanIocod/chat_backend/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Send message

func SendMessage(c *gin.Context) {
	// Get user ID from JWT (middleware)
	userID := c.MustGet("userID").(string)
	senderID, _ := primitive.ObjectIDFromHex(userID)

	var req models.MessageRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Invalid request",
			Data:         nil,
		})
		return
	}

	// Convert receiver ID to ObjectID
	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Invalid receiver ID",
			Data:         nil,
		})
		return
	}

	ctx := context.Background()
	userCollection := utils.DB.Collection("users")
	userCount, err := userCollection.CountDocuments(ctx, bson.M{"_id": receiverID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Error checking receiver",
			Data:         nil,
		})
		return
	}
	if userCount == 0 {
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Receiver does not exist",
			Data:         nil,
		})
		return
	}

	// Create message
	message := models.Message{
		ID:         primitive.NewObjectID(),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    req.Content,
		Seen:       false,
		CreatedAt:  time.Now(),
	}

	// Save to MongoDB
	_, err = utils.DB.Collection("messages").InsertOne(ctx, message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to send message",
			Data:         nil,
		})
		return
	}

	// Trigger Pusher event
	eventData := gin.H{
		"message": message,
		"type":    "new-message",
	}

	// Trigger to both sender and receiver channels
	channelName := "private-chat-" + receiverID.Hex()
	err = utils.PusherClient.Trigger(channelName, "message", eventData)
	if err != nil {
		log.Printf("[ERROR] Failed to trigger Pusher event: %v", err)
		// Don't return error to client as message is already saved
	}

	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Message sent successfully",
		Data:         message,
	})
}

// Get messages between logged-in user and another user
func GetMessages(c *gin.Context) {
	currentUser := utils.ObjectIDFromHex(c.GetString("userID"))
	otherID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	filter := bson.M{
		"$or": []bson.M{
			{"senderId": currentUser, "receiverId": otherID},
			{"senderId": otherID, "receiverId": currentUser},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"createdAt": 1}) // sort by time
	cursor, err := utils.DB.Collection("messages").Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse messages"})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Messages fetched successfully",
		Data:         messages,
	})
}

// Mark messages as seen
func MarkMessagesSeen(c *gin.Context) {
	currentUser := utils.ObjectIDFromHex(c.GetString("userID"))
	otherID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Updated filter to match exact field names from Message model
	filter := bson.M{
		"senderID":   otherID,     // Make sure these match your Message struct field names
		"receiverID": currentUser, // Make sure these match your Message struct field names
		"seen":       false,
	}

	update := bson.M{
		"$set": bson.M{"seen": true},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := utils.DB.Collection("messages").UpdateMany(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to mark messages as seen: " + err.Error(),
			Data:         nil,
		})
		return
	}

	// Add response with modified count for debugging
	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Messages marked as seen",
		Data: gin.H{
			"modifiedCount": result.ModifiedCount,
			"matchedCount":  result.MatchedCount,
		},
	})
}

// delete - chat history
func DeleteChatHistory(c *gin.Context) {
	currentUser := utils.ObjectIDFromHex(c.GetString("userID"))
	otherID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Invalid user ID",
			Data:         nil,
		})
		return
	}

	filter := bson.M{
		"$or": []bson.M{
			{"senderId": currentUser, "receiverId": otherID},
			{"senderId": otherID, "receiverId": currentUser},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := utils.DB.Collection("messages").DeleteMany(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to delete chat history",
			Data:         nil,
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Chat history deleted successfully",
		Data: gin.H{
			"deletedCount": result.DeletedCount,
			"matchedCount": result.DeletedCount,
		},
	})
}

func GetChatByID(c *gin.Context) {
	chatID := c.Query("id")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Chat ID is required",
			Data:         nil,
		})
		return
	}

	currentUserID := c.GetString("userID")
	log.Printf("[INFO] Fetching chat by ID: %s for user: %s", chatID, currentUserID)

	// Convert string IDs to ObjectIDs
	currentUserObjID := utils.ObjectIDFromHex(currentUserID)
	chatObjID, err := primitive.ObjectIDFromHex(chatID)
	if err != nil {
		log.Printf("[ERROR] Invalid chat ID format: %v", err)
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Invalid chat ID format",
			Data:         nil,
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build aggregation pipeline
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"_id": chatObjID,
				"$or": []bson.M{
					{"senderID": currentUserObjID},
					{"receiverID": currentUserObjID},
				},
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "senderID",
				"foreignField": "_id",
				"as":           "sender",
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "receiverID",
				"foreignField": "_id",
				"as":           "receiver",
			},
		},
		{
			"$unwind": "$sender",
		},
		{
			"$unwind": "$receiver",
		},
		{
			"$project": bson.M{
				"_id":       1,
				"content":   1,
				"seen":      1,
				"createdAt": 1,
				"sender": bson.M{
					"_id":      1,
					"username": 1,
					"email":    1,
				},
				"receiver": bson.M{
					"_id":      1,
					"username": 1,
					"email":    1,
				},
			},
		},
	}

	// Execute aggregation
	cursor, err := utils.DB.Collection("messages").Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("[ERROR] Failed to execute aggregation: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to fetch chat details",
			Data:         nil,
		})
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		log.Printf("[ERROR] Failed to decode results: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to process chat details",
			Data:         nil,
		})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			ResponseCode: http.StatusNotFound,
			Message:      "Chat not found or you don't have access to it",
			Data:         nil,
		})
		return
	}

	// Return the first (and should be only) result
	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Chat fetched successfully",
		Data:         results[0],
	})
}
