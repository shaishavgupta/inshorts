package controllers

import (
	"time"

	"news-inshorts/src/infra"
	"news-inshorts/src/models"
	"news-inshorts/src/repositories"
	"news-inshorts/src/types"

	"github.com/gofiber/fiber/v2"
)

// UserInteractionController handles user interaction-related HTTP requests
type UserInteractionController struct {
	userEventRepo repositories.UserEventRepository
	logger        infra.Logger
}

// NewUserInteractionController creates a new instance of UserInteractionController
func NewUserInteractionController(userEventRepo repositories.UserEventRepository) *UserInteractionController {
	return &UserInteractionController{
		userEventRepo: userEventRepo,
		logger:        infra.GetLogger(),
	}
}

// RecordInteraction handles POST /api/v1/interactions
// Records user interaction events with news articles
func (uic *UserInteractionController) RecordInteraction(c *fiber.Ctx) error {
	var req types.RecordInteractionRequest

	// Parse and validate request body
	if err := c.BodyParser(&req); err != nil {
		uic.logger.Error("Failed to parse request body", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id field is required",
		})
	}

	if req.ArticleID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "article_id field is required",
		})
	}

	if req.EventType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "event_type field is required",
		})
	}

	// Validate event type
	if req.EventType != "view" && req.EventType != "click" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "event_type must be either 'view' or 'click'",
		})
	}

	// Validate location
	if req.Location.Latitude < -90 || req.Location.Latitude > 90 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Latitude must be between -90 and 90",
		})
	}

	if req.Location.Longitude < -180 || req.Location.Longitude > 180 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Longitude must be between -180 and 180",
		})
	}

	uic.logger.Info("Recording user interaction", map[string]interface{}{
		"user_id":    req.UserID,
		"article_id": req.ArticleID,
		"event_type": req.EventType,
	})

	// Create UserEvent model
	event := &models.UserEvent{
		UserID:    req.UserID,
		ArticleID: req.ArticleID,
		EventType: req.EventType,
		Timestamp: time.Now(),
		Latitude:  req.Location.Latitude,
		Longitude: req.Location.Longitude,
	}

	// Call UserEventRepository to create the event
	err := uic.userEventRepo.Create(event)
	if err != nil {
		uic.logger.Error("Failed to record user interaction", err, map[string]interface{}{
			"user_id":    req.UserID,
			"article_id": req.ArticleID,
			"event_type": req.EventType,
		})

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to record interaction",
		})
	}

	// Return success response with event ID
	response := types.RecordInteractionResponse{
		Success: true,
		EventID: event.ID,
	}

	uic.logger.Info("User interaction recorded successfully", map[string]interface{}{
		"event_id":   event.ID,
		"user_id":    req.UserID,
		"article_id": req.ArticleID,
		"event_type": req.EventType,
	})

	return c.Status(fiber.StatusOK).JSON(response)
}
