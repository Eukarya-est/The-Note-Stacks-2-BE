package handlers

import (
	"fmt"
	"net/http"
	"note-stacks-backend/utils"
	"time"

	"note-stacks-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===== CALENDAR EVENT HANDLERS =====

// CreateCalendarEvent handles POST requests to create a new calendar event
// POST /api/events
// Request body: CreateCalendarEventRequest (title, description, label, event_date)
// Response: CalendarEvent with generated ID and timestamps
func (h *Handler) CreateCalendarEvent(c *gin.Context) {
	var req models.CreateCalendarEventRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate event label
	if !isValidEventLabel(req.Label) {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid event label. Must be one of: reading, repeat, shadowing, solve, writing, development")
		return
	}

	// Validate date format (YYYY-MM-DD)
	if _, err := time.Parse("2006-01-02", req.EventDate); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
		return
	}

	now := time.Now()
	event := &models.CalendarEvent{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Label:       req.Label,
		EventDate:   req.EventDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.CreateCalendarEvent(c.Request.Context(), event); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to create event: %v", err))
		return
	}

	utils.RespondWithJSON(c, http.StatusCreated, event)
}

// GetCalendarEvent handles GET requests to retrieve a calendar event by ID
// GET /api/events/:id
// Response: CalendarEvent with all fields
func (h *Handler) GetCalendarEvent(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Event ID is required")
		return
	}

	event, err := h.repo.GetCalendarEventByID(c.Request.Context(), id)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Event not found")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, event)
}

// UpdateCalendarEvent handles PUT requests to update an existing calendar event
// PUT /api/events/:id
// Request body: UpdateCalendarEventRequest (optional: title, description, label, event_date)
// Response: Updated CalendarEvent
func (h *Handler) UpdateCalendarEvent(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Event ID is required")
		return
	}

	var req models.UpdateCalendarEventRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	existingEvent, err := h.repo.GetCalendarEventByID(c.Request.Context(), id)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Event not found")
		return
	}

	// Update fields if provided
	if req.Title != "" {
		existingEvent.Title = req.Title
	}
	if req.Description != "" {
		existingEvent.Description = req.Description
	}
	if req.Label != "" {
		if !isValidEventLabel(req.Label) {
			utils.RespondWithError(c, http.StatusBadRequest, "Invalid event label")
			return
		}
		existingEvent.Label = req.Label
	}
	if req.EventDate != "" {
		if _, err := time.Parse("2006-01-02", req.EventDate); err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}
		existingEvent.EventDate = req.EventDate
	}

	existingEvent.UpdatedAt = time.Now()

	if err := h.repo.UpdateCalendarEvent(c.Request.Context(), existingEvent); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to update event")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, existingEvent)
}

// DeleteCalendarEvent handles DELETE requests to remove a calendar event
// DELETE /api/events/:id
// Response: 204 No Content on success
func (h *Handler) DeleteCalendarEvent(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Event ID is required")
		return
	}

	if err := h.repo.DeleteCalendarEvent(c.Request.Context(), id); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to delete event")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAllCalendarEvents handles GET requests to retrieve all calendar events
// GET /api/events
// Query params:
//   - date (optional): Filter by specific date (YYYY-MM-DD)
//   - start_date (optional): Filter events from this date onwards
//   - end_date (optional): Filter events up to this date
//   - label (optional): Filter by event label
//
// Response: Array of CalendarEvent objects
func (h *Handler) GetAllCalendarEvents(c *gin.Context) {
	events, err := h.repo.GetAllCalendarEvents(c.Request.Context())
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve events")
		return
	}

	// Apply filters if provided
	date := c.Query("date")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	label := c.Query("label")

	filteredEvents := filterEvents(events, date, startDate, endDate, label)

	utils.RespondWithJSON(c, http.StatusOK, filteredEvents)
}

// Helper function to validate event labels
func isValidEventLabel(label string) bool {
	validLabels := map[string]bool{
		"reading":     true,
		"repeat":      true,
		"shadowing":   true,
		"solve":       true,
		"writing":     true,
		"development": true,
	}
	return validLabels[label]
}

// Helper function to filter events based on query parameters
func filterEvents(events []*models.CalendarEvent, date, startDate, endDate, label string) []*models.CalendarEvent {
	filtered := make([]*models.CalendarEvent, 0)

	for _, event := range events {
		// Filter by specific date
		if date != "" && event.EventDate != date {
			continue
		}

		// Filter by date range
		if startDate != "" && event.EventDate < startDate {
			continue
		}
		if endDate != "" && event.EventDate > endDate {
			continue
		}

		// Filter by label
		if label != "" && event.Label != label {
			continue
		}

		filtered = append(filtered, event)
	}

	return filtered
}
