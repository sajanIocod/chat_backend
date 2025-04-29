package controllers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sajanbaisil/chat_backend/models"
	"github.com/sajanbaisil/chat_backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var user models.User
	c.BindJSON(&user)

	collection := utils.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if user exists
	count, _ := collection.CountDocuments(ctx, bson.M{"email": user.Email})
	if count > 0 {
		c.JSON(400, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
	user.Password = string(hash)

	res, err := collection.InsertOne(ctx, user)
	if err != nil {
		c.JSON(500, gin.H{"error": "Registration failed"})
		return
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		user.ID = oid
	}
	c.JSON(200, user)
}
