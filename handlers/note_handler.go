package handlers

import (
	"net/http"
	"note-stacks-backend/utils"
	"time"

	"note-stacks-backend/models"
	"note-stacks-backend/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for all entities
type Handler struct {
	repo repository.Repository
}

// NewHandler creates a new instance of Handler
// repo: The repository implementation for data access
func NewHandler(repo repository.Repository) *Handler {
	return &Handler{
		repo: repo,
	}
}

// ===== NOTE HANDLERS =====

// CreateNote handles POST requests to create a new note
// POST /api/notes
func (h *Handler) CreateNote(c *gin.Context) {
	var req models.CreateNoteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	now := time.Now()
	note := &models.Note{
		ID:       uuid.New().String(),
		Title:    req.Title,
		Filename: req.Filename,
		Filepath: req.Filepath,
		Content:  req.Content,
		Cover:    req.Cover,
		Num:      req.Num,
		Revision: req.Revision,
		Created:  now,
		Revised:  now,
		Display:  req.Display,
	}

	// Set defaults
	if note.Revision == "" {
		note.Revision = "v1.0"
	}
	if note.Num == 0 {
		note.Num = 1
	}

	if err := h.repo.CreateNote(c.Request.Context(), note); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create note")
		return
	}

	utils.RespondWithJSON(c, http.StatusCreated, note)
}

// GetNote handles GET requests to retrieve a note by ID
// GET /api/notes/:id
func (h *Handler) GetNote(c *gin.Context) {
	id := c.Param("id") // ✅ FIX: Get actual ID from URL

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Note ID is required")
		return
	}

	note, err := h.repo.GetNoteByID(c.Request.Context(), id) // ✅ FIX: Use variable
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Note not found")
		return
	}

	// 🛠️ #15.2: Check display field
	if !note.Display {
		utils.RespondWithError(c, http.StatusNotFound, "Note not found")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, note)
}

// UpdateNote handles PUT requests to update an existing note
// PUT /api/notes/:id
func (h *Handler) UpdateNote(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Note ID is required")
		return
	}

	var req models.UpdateNoteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	existingNote, err := h.repo.GetNoteByID(c.Request.Context(), id)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Note not found")
		return
	}

	// Update fields if provided
	if req.Title != "" {
		existingNote.Title = req.Title
	}
	if req.Filename != "" {
		existingNote.Filename = req.Filename
	}
	if req.Filepath != "" {
		existingNote.Filepath = req.Filepath
	}
	if req.Content != "" {
		existingNote.Content = req.Content
	}
	if req.Cover != "" {
		existingNote.Cover = req.Cover
	}
	if req.Num != 0 {
		existingNote.Num = req.Num
	}
	if req.Revision != "" {
		existingNote.Revision = req.Revision
	}
	existingNote.Display = req.Display

	existingNote.Revised = time.Now()

	if err := h.repo.UpdateNote(c.Request.Context(), existingNote); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to update note")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, existingNote)
}

// DeleteNote handles DELETE requests to remove a note
// DELETE /api/notes/:id
func (h *Handler) DeleteNote(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Note ID is required")
		return
	}

	if err := h.repo.DeleteNote(c.Request.Context(), id); err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Note not found")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetNotesByCover handles GET requests to retrieve all notes in a category/cover
// GET /api/covers/:id/notes
func (h *Handler) GetNotesByCover(c *gin.Context) {
	coverID := c.Param("id")

	// ✅ Verify cover exists and is valid
	cover, err := h.repo.GetCoverByID(c.Request.Context(), coverID)
	if err != nil || !cover.Valid {
		utils.RespondWithError(c, http.StatusNotFound, "Cover not found")
		return
	}

	notes, err := h.repo.GetNotesByCover(c.Request.Context(), coverID)
	utils.RespondWithJSON(c, http.StatusOK, notes)
}

// SearchNotes handles GET requests to search notes by title
// GET /api/notes/search?q=query
func (h *Handler) SearchNotes(c *gin.Context) {
	query := c.Query("q")

	if query == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Search query is required")
		return
	}

	notes, err := h.repo.SearchNotes(c.Request.Context(), query)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to search notes")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, notes)
}

// GetAllNotes handles GET requests to retrieve all notes
// GET /api/notes
func (h *Handler) GetAllNotes(c *gin.Context) {
	notes, err := h.repo.GetAllNotes(c.Request.Context())
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve notes")
		return
	}

	// 🛠️ #15.2: Filter by display
	displayedNotes := make([]*models.Note, 0)
	for _, note := range notes {
		if note.Display {
			displayedNotes = append(displayedNotes, note)
		}
	}

	utils.RespondWithJSON(c, http.StatusOK, displayedNotes)
}

// ===== COVER HANDLERS =====

// CreateCover handles POST requests to create a new category/cover
// POST /api/covers
func (h *Handler) CreateCover(c *gin.Context) {
	var req models.CreateCoverRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	cover := &models.Cover{
		ID:       uuid.New().String(),
		Category: req.Category,
		Valid:    req.Valid,
		Created:  time.Now(),
	}

	if err := h.repo.CreateCover(c.Request.Context(), cover); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create cover")
		return
	}

	utils.RespondWithJSON(c, http.StatusCreated, cover)
}

// GetCover handles GET requests to retrieve a cover by ID
// GET /api/covers/:id
func (h *Handler) GetCover(c *gin.Context) {
	id := c.Param("id") // ✅ FIX: Get actual ID from URL

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Cover ID is required")
		return
	}

	cover, err := h.repo.GetCoverByID(c.Request.Context(), id) // ✅ FIX: Use variable
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Cover not found")
		return
	}

	// 🛠️ #15.2: Filter by Valid
	if !cover.Valid {
		utils.RespondWithError(c, http.StatusNotFound, "Cover not found")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, cover)
}

// UpdateCover handles PUT requests to update an existing cover
// PUT /api/covers/:id
func (h *Handler) UpdateCover(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Cover ID is required")
		return
	}

	var req models.UpdateCoverRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	existingCover, err := h.repo.GetCoverByID(c.Request.Context(), id)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Cover not found")
		return
	}

	if req.Category != "" {
		existingCover.Category = req.Category
	}
	existingCover.Valid = req.Valid

	if err := h.repo.UpdateCover(c.Request.Context(), existingCover); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to update cover")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, existingCover)
}

// DeleteCover handles DELETE requests to remove a cover
// DELETE /api/covers/:id
func (h *Handler) DeleteCover(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Cover ID is required")
		return
	}

	if err := h.repo.DeleteCover(c.Request.Context(), id); err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Cover not found")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAllCovers handles GET requests to retrieve all covers
// GET /api/covers
func (h *Handler) GetAllCovers(c *gin.Context) {
	covers, err := h.repo.GetAllCovers(c.Request.Context())
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve covers")
		return
	}

	// 🛠️ #15.2: Filter by valid
	validCovers := make([]*models.Cover, 0)
	for _, cover := range covers {
		if cover.Valid {
			validCovers = append(validCovers, cover)
		}
	}

	utils.RespondWithJSON(c, http.StatusOK, validCovers)
}

// ===== NOTE VIEW HANDLERS =====

// TrackView handles POST requests to track a note view
// POST /api/views
func (h *Handler) TrackView(c *gin.Context) {
	var req models.TrackViewRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	view := &models.NoteView{
		ID:        uuid.New().String(),
		NoteID:    req.NoteID,
		ViewedAt:  time.Now(),
		SessionID: req.SessionID,
	}

	if err := h.repo.TrackNoteView(c.Request.Context(), view); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to track view")
		return
	}

	utils.RespondWithJSON(c, http.StatusCreated, view)
}

// GetViewCount handles GET requests to get view count for a note
// GET /api/notes/:id/views/count
func (h *Handler) GetViewCount(c *gin.Context) {
	noteID := c.Param("id")

	if noteID == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Note ID is required")
		return
	}

	count, err := h.repo.GetViewCountByNote(c.Request.Context(), noteID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get view count")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, gin.H{
		"note_id": noteID,
		"count":   count,
	})
}
