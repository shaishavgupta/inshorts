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
func (uic *UserInteractionController) RecordInteraction(c *fiber.Ctx) error {
	var req types.RecordInteractionRequest

	if err := c.BodyParser(&req); err != nil {
		uic.logger.Error("Failed to parse request body", err, map[string]interface{}{
			"path": c.Path(),
		})
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	uic.logger.Info("Recording user interaction", map[string]interface{}{
		"user_id":    req.UserID,
		"article_id": req.ArticleID,
		"event_type": req.EventType,
	})

	event := &models.UserEvent{
		UserID:    req.UserID,
		ArticleID: req.ArticleID,
		EventType: req.EventType,
		Timestamp: time.Now(),
		Latitude:  req.Location.Latitude,
		Longitude: req.Location.Longitude,
	}

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
