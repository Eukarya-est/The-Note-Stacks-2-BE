package handlers

import (
	"net/http"
	"strconv"

	"note-stacks-backend/repository"
	"note-stacks-backend/utils"

	"github.com/gin-gonic/gin"
)

// AdvancedSearch handles GET requests for advanced search with filters
// GET /api/notes/search/advanced?q=query&cover=id&from=date&to=date&sort=field&order=asc
func (h *Handler) AdvancedSearch(c *gin.Context) {
	// Check if we have an Elasticsearch-enabled repository
	compositeRepo, ok := h.repo.(*repository.CompositeRepository)
	if !ok {
		// Fallback to simple search
		h.SearchNotes(c)
		return
	}

	// Parse query parameters
	params := repository.SearchParams{
		Query:     c.Query("q"),
		CoverID:   c.Query("cover"),
		DateFrom:  c.Query("from"),
		DateTo:    c.Query("to"),
		SortBy:    c.DefaultQuery("sort", "created"),
		SortOrder: c.DefaultQuery("order", "desc"),
		Size:      100,
		From:      0,
	}

	// Parse pagination if provided
	if size := c.Query("size"); size != "" {
		if s, err := strconv.Atoi(size); err == nil && s > 0 {
			params.Size = s
		}
	}

	if from := c.Query("offset"); from != "" {
		if f, err := strconv.Atoi(from); err == nil && f >= 0 {
			params.From = f
		}
	}

	// Get Elasticsearch repository from composite
	// This is a temporary solution - in production, you'd want better type handling
	esRepo := compositeRepo.GetElasticsearchRepo()
	if esRepo == nil {
		// Fallback to simple search
		h.SearchNotes(c)
		return
	}

	notes, err := esRepo.AdvancedSearch(c.Request.Context(), params)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to perform advanced search")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, notes)
}

// ReindexNotes handles POST requests to reindex all notes in Elasticsearch
// POST /api/admin/reindex
// This should be protected with authentication in production
func (h *Handler) ReindexNotes(c *gin.Context) {
	// Check if we have an Elasticsearch-enabled repository
	compositeRepo, ok := h.repo.(*repository.CompositeRepository)
	if !ok {
		utils.RespondWithError(c, http.StatusBadRequest, "Elasticsearch is not enabled")
		return
	}

	esRepo := compositeRepo.GetElasticsearchRepo()
	if esRepo == nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Elasticsearch is not enabled")
		return
	}

	// Reindex all notes
	if err := esRepo.ReindexAllNotes(c.Request.Context()); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to reindex notes")
		return
	}

	utils.RespondWithJSON(c, http.StatusOK, gin.H{
		"message": "Reindexing completed successfully",
	})
}
