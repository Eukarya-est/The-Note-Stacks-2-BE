package models

import (
	"time"
)

// Note represents a note/markdown document in the system
// Previously called "notes" table in Supabase
type Note struct {
	ID       string    `json:"id"`                 // UUID
	Title    string    `json:"title"`              // Note title
	Filename string    `json:"filename"`           // Original filename
	Filepath string    `json:"filepath"`           // Path to markdown file in storage (optional)
	Content  string    `json:"content"`            // Markdown content
	Cover    string    `json:"cover"`              // Category/cover ID (foreign key)
	Category *Cover    `json:"category,omitempty"` // Populated category object (not stored)
	Num      int       `json:"num"`                // Display order number
	Revision string    `json:"revision"`           // Version string (e.g., "v1.0")
	Created  time.Time `json:"created"`            // Creation timestamp
	Revised  time.Time `json:"revised"`            // Last revision timestamp
	Display  bool      `json:"display"`            // Whether to display this note
}

// Cover represents a category/cover for organizing notes
// Previously called "cover" table in Supabase
type Cover struct {
	ID       string    `json:"id"`       // UUID
	Category string    `json:"category"` // Category name
	Valid    bool      `json:"valid"`    // Whether this category is active
	Created  time.Time `json:"created"`  // Creation timestamp
}

// NoteView represents a tracked view of a note
// Previously called "note_views" table in Supabase
type NoteView struct {
	ID        string    `json:"id"`         // UUID
	NoteID    string    `json:"note_id"`    // Note being viewed
	ViewedAt  time.Time `json:"viewed_at"`  // When it was viewed
	SessionID string    `json:"session_id"` // Session identifier
}

// Request types for creating/updating entities

// CreateNoteRequest represents the payload for creating a new note
type CreateNoteRequest struct {
	Title    string `json:"title" binding:"required"`
	Filename string `json:"filename"`
	Filepath string `json:"filepath"` // Optional path to markdown file in storage
	Content  string `json:"content" binding:"required"`
	Cover    string `json:"cover" binding:"required"` // Category ID
	Num      int    `json:"num"`
	Revision string `json:"revision"`
	Display  bool   `json:"display"`
}

// UpdateNoteRequest represents the payload for updating an existing note
type UpdateNoteRequest struct {
	Title    string `json:"title"`
	Filename string `json:"filename"`
	Filepath string `json:"filepath"`
	Content  string `json:"content"`
	Cover    string `json:"cover"`
	Num      int    `json:"num"`
	Revision string `json:"revision"`
	Display  bool   `json:"display"`
}

// CreateCoverRequest represents the payload for creating a new category
type CreateCoverRequest struct {
	Category string `json:"category" binding:"required"`
	Valid    bool   `json:"valid"`
}

// UpdateCoverRequest represents the payload for updating a category
type UpdateCoverRequest struct {
	Category string `json:"category"`
	Valid    bool   `json:"valid"`
}

// TrackViewRequest represents the payload for tracking a note view
type TrackViewRequest struct {
	NoteID    string `json:"note_id" binding:"required"`
	SessionID string `json:"session_id" binding:"required"`
}
