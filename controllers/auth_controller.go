package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sajanIocod/chat_backend/models"
	"github.com/sajanIocod/chat_backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var user models.User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(400, models.Response{
			ResponseCode: 400,
			Message:      "Invalid request data",
			Data:         nil,
		})
		return
	}

	collection := utils.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Generate a new ObjectID for the user
	user.ID = primitive.NewObjectID()

	// Check if user exists
	existingCount, _ := collection.CountDocuments(ctx, bson.M{"email": user.Email})
	if existingCount > 0 {
		c.JSON(400, models.Response{
			ResponseCode: 400,
			Message:      "Email already registered",
			Data:         nil,
		})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, models.Response{
			ResponseCode: 500,
			Message:      "Error hashing password",
			Data:         nil,
		})
		return
	}
	user.Password = string(hash)
	// Insert user
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		c.JSON(500, models.Response{
			ResponseCode: 500,
			Message:      "Registration failed: " + err.Error(),
			Data:         nil,
		})
		return
	}

	// Generate JWT token
	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID.Hex())
	if err != nil {
		c.JSON(500, models.Response{
			ResponseCode: 500,
			Message:      "Error generating token",
			Data:         nil,
		})
		return
	}

	// Don't send password in response
	userResp := models.UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		Token:    "Bearer " + token,
	}
	c.JSON(200, models.Response{
		ResponseCode: 200,
		Message:      "User registered successfully",
		Data:         userResp,
	})
}

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}

	collection := utils.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&user)
	if err != nil {
		c.JSON(401, models.Response{
			ResponseCode: 401,
			Message:      "Invalid email or password",
			Data:         nil,
		})
		return
	}
	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		c.JSON(401, models.Response{
			ResponseCode: 401,
			Message:      "Invalid email or password",
			Data:         nil,
		})
		return
	}
	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID.Hex())
	if err != nil {
		c.JSON(500, models.Response{
			ResponseCode: 500,
			Message:      "Error generating token",
			Data:         nil,
		})
		return
	}
	// Don't send password in response
	userResp := models.UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		Token:    "Bearer " + token,
	}
	c.JSON(200, models.Response{
		ResponseCode: 200,
		Message:      "Login successful",
		Data:         userResp,
	})

}

func Profile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(500, gin.H{"error": "User ID not found in context"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Access granted",
		"userID":  userID,
	})
}

func Logout(c *gin.Context) {
	tokenString := c.GetHeader("Authorization") // Bearer <token>
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Authorization header is required",
			Data:         nil,
		})
		return
	}
	token := strings.TrimPrefix(tokenString, "Bearer ")

	// Save to blacklist
	blacklisted := models.BlacklistedToken{
		Token:     token,
		ExpiredAt: time.Now().Add(time.Hour * 24), // depends on token expiry
	}
	_, err := utils.DB.Collection("blacklisted_tokens").InsertOne(context.TODO(), blacklisted)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to blacklist token",
			Data:         nil,
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Logout successful",
		Data:         nil,
	})
}
