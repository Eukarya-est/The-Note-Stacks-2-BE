package models

import "time"

// CalendarEvent represents an event on the shared calendar
// Previously called "calendar_events" table in Supabase
type CalendarEvent struct {
	ID          string    `json:"id"`          // UUID
	Title       string    `json:"title"`       // Event title
	Description string    `json:"description"` // Event description (optional)
	Label       string    `json:"label"`       // Event type/label (reading, repeat, etc.)
	EventDate   string    `json:"event_date"`  // Event date (YYYY-MM-DD)
	CreatedAt   time.Time `json:"created_at"`  // Creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`  // Last update timestamp
}

// CreateCalendarEventRequest represents the payload for creating a calendar event
type CreateCalendarEventRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Label       string `json:"label" binding:"required"`
	EventDate   string `json:"event_date" binding:"required"` // YYYY-MM-DD format
}

// UpdateCalendarEventRequest represents the payload for updating a calendar event
type UpdateCalendarEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Label       string `json:"label"`
	EventDate   string `json:"event_date"`
}
