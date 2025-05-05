package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sajanIocod/chat_backend/controllers"
	"github.com/sajanIocod/chat_backend/utils"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Public routes
	r.POST("/register", controllers.Register)
	r.POST("/login", controllers.Login)

	// Protected routes
	auth := r.Group("/api")
	auth.Use(utils.JWTAuthMiddleware())
	{
		// auth.GET("/profile", controllers.Profile) // Example
		auth.GET("/users", controllers.GetUsers) // To be created
		auth.POST("/messages", controllers.SendMessage)
		auth.GET("/messages/:userId", controllers.GetMessages)
		auth.POST("/messages/markseen/:userId", controllers.MarkMessagesSeen)
		auth.POST("/suggestions", controllers.GetReplySuggestions)

	}

	return r
}
