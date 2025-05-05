package controllers

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sajanIocod/chat_backend/models"
	"google.golang.org/genai"
)

func GetReplySuggestions(c *gin.Context) {
	var body struct {
		History []models.Message `json:"history" binding:"required,dive"`
	}

	// Parse request body
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Printf("[ERROR] Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Invalid request: History array is required with valid messages",
			Data:         nil,
		})
		return
	}

	// Validate history
	if len(body.History) == 0 {
		log.Printf("[ERROR] Empty chat history received")
		c.JSON(http.StatusBadRequest, models.Response{
			ResponseCode: http.StatusBadRequest,
			Message:      "Chat history cannot be empty",
			Data:         nil,
		})
		return
	}

	// Get API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Printf("[ERROR] GEMINI_API_KEY environment variable not set")
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Missing Gemini API key",
			Data:         nil,
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize Gemini client
	log.Printf("[INFO] Initializing Gemini client")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Printf("[ERROR] Failed to create Gemini client: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to initialize Gemini client",
			Data:         nil,
		})
		return
	}

	// Format chat history for prompt
	var formattedHistory strings.Builder
	for _, msg := range body.History {
		role := "User"
		if msg.SenderID == msg.ReceiverID {
			role = "Bot"
		}
		formattedHistory.WriteString(role + ": " + msg.Content + "\n")
	}
	prompt := "Based on this chat history:\n" + formattedHistory.String() +
		"\n\nProvide exactly 3 short, natural reply suggestions. Format them as a numbered list (1., 2., 3.)"
	// Prepare model and prompt
	result, err := client.Models.GenerateContent(ctx,
		"gemini-2.0-flash",
		genai.Text(prompt),
		nil)

	log.Printf("[INFO] Sending prompt to Gemini API with model: gemini-1.0-pro")
	// Configure generation parameters

	// Generate content
	if err != nil {
		log.Printf("[ERROR] Failed to generate content: %v", err)
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "Failed to generate suggestions",
			Data:         nil,
		})
		return
	}

	if result.Text() == "" {
		log.Printf("[ERROR] No text content found in response")
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "No text content in response",
			Data:         nil,
		})
		return
	}

	// Extract suggestions
	suggestions := []string{}
	lines := strings.Split(result.Text(), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove number prefixes and clean up
		for _, prefix := range []string{"1.", "2.", "3.", "1)", "2)", "3)"} {
			line = strings.TrimPrefix(line, prefix)
		}
		line = strings.TrimSpace(line)
		if line != "" {
			suggestions = append(suggestions, line)
		}
	}

	if len(suggestions) == 0 {
		log.Printf("[ERROR] No suggestions extracted from response")
		c.JSON(http.StatusInternalServerError, models.Response{
			ResponseCode: http.StatusInternalServerError,
			Message:      "No suggestions found in response",
			Data:         nil,
		})
		return
	}

	log.Printf("[INFO] Successfully generated %d suggestions", len(suggestions))

	// Return suggestions
	c.JSON(http.StatusOK, models.Response{
		ResponseCode: http.StatusOK,
		Message:      "Suggestions generated successfully",
		Data: gin.H{
			"suggestions": suggestions,
		},
	})
}
