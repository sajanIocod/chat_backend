package controllers

import (
	"context"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Convert receiver ID to ObjectID
	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receiver ID"})
		return
	}

	ctx := context.Background()
	userCollection := utils.DB.Collection("users")
	userCount, err := userCollection.CountDocuments(ctx, bson.M{"_id": receiverID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking receiver"})
		return
	}
	if userCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Receiver does not exist"})
		return
	}

	message := bson.M{
		"senderId":   senderID,
		"receiverId": receiverID,
		"content":    req.Content,
		"seen":       false,
		"createdAt":  time.Now(),
	}

	_, err = utils.DB.Collection("messages").InsertOne(ctx, message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent"})
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

