package repository

import (
	"context"
	"note-stacks-backend/models"
)

// CompositeRepository combines Redis (for data storage) and Elasticsearch (for search)
type CompositeRepository struct {
	redis *RedisRepository
	es    *ElasticsearchRepository
}

// NewCompositeRepository creates a new composite repository
func NewCompositeRepository(redisRepo *RedisRepository, esRepo *ElasticsearchRepository) *CompositeRepository {
	return &CompositeRepository{
		redis: redisRepo,
		es:    esRepo,
	}
}

// ===== NOTE OPERATIONS =====

// CreateNote stores a note in Redis and indexes it in Elasticsearch
func (c *CompositeRepository) CreateNote(ctx context.Context, note *models.Note) error {
	// Store in Redis
	if err := c.redis.CreateNote(ctx, note); err != nil {
		return err
	}

	// Index in Elasticsearch (non-blocking, log errors)
	go func() {
		if err := c.es.IndexNote(context.Background(), note); err != nil {
			// In production, use proper logging
			println("Failed to index note in Elasticsearch:", err.Error())
		}
	}()

	return nil
}

// GetNoteByID retrieves a note from Redis
func (c *CompositeRepository) GetNoteByID(ctx context.Context, id string) (*models.Note, error) {
	return c.redis.GetNoteByID(ctx, id)
}

// UpdateNote updates a note in Redis and reindexes it in Elasticsearch
func (c *CompositeRepository) UpdateNote(ctx context.Context, note *models.Note) error {
	// Update in Redis
	if err := c.redis.UpdateNote(ctx, note); err != nil {
		return err
	}

	// Reindex in Elasticsearch (non-blocking)
	go func() {
		if err := c.es.IndexNote(context.Background(), note); err != nil {
			println("Failed to reindex note in Elasticsearch:", err.Error())
		}
	}()

	return nil
}

// DeleteNote removes a note from Redis and Elasticsearch
func (c *CompositeRepository) DeleteNote(ctx context.Context, id string) error {
	// Delete from Redis
	if err := c.redis.DeleteNote(ctx, id); err != nil {
		return err
	}

	// Delete from Elasticsearch (non-blocking)
	go func() {
		if err := c.es.DeleteNoteFromIndex(context.Background(), id); err != nil {
			println("Failed to delete note from Elasticsearch:", err.Error())
		}
	}()

	return nil
}

// GetNotesByCover retrieves notes by cover from Redis
func (c *CompositeRepository) GetNotesByCover(ctx context.Context, coverID string) ([]*models.Note, error) {
	return c.redis.GetNotesByCover(ctx, coverID)
}

// SearchNotes uses Elasticsearch for full-text search
func (c *CompositeRepository) SearchNotes(ctx context.Context, query string) ([]*models.Note, error) {
	if c.es != nil {
		return c.es.SearchNotes(ctx, query)
	}
	// Fallback to Redis search if Elasticsearch is not available
	return c.redis.SearchNotes(ctx, query)
}

// GetAllNotes retrieves all notes from Redis
func (c *CompositeRepository) GetAllNotes(ctx context.Context) ([]*models.Note, error) {
	return c.redis.GetAllNotes(ctx)
}

// ===== COVER OPERATIONS =====

// CreateCover stores a cover in Redis
func (c *CompositeRepository) CreateCover(ctx context.Context, cover *models.Cover) error {
	return c.redis.CreateCover(ctx, cover)
}

// GetCoverByID retrieves a cover from Redis
func (c *CompositeRepository) GetCoverByID(ctx context.Context, id string) (*models.Cover, error) {
	return c.redis.GetCoverByID(ctx, id)
}

// UpdateCover updates a cover in Redis
func (c *CompositeRepository) UpdateCover(ctx context.Context, cover *models.Cover) error {
	return c.redis.UpdateCover(ctx, cover)
}

// DeleteCover removes a cover from Redis
func (c *CompositeRepository) DeleteCover(ctx context.Context, id string) error {
	return c.redis.DeleteCover(ctx, id)
}

// GetAllCovers retrieves all covers from Redis
func (c *CompositeRepository) GetAllCovers(ctx context.Context) ([]*models.Cover, error) {
	return c.redis.GetAllCovers(ctx)
}

// ===== CALENDAR EVENT OPERATIONS =====

// CreateCalendarEvent stores a calendar event in Redis
func (c *CompositeRepository) CreateCalendarEvent(ctx context.Context, event *models.CalendarEvent) error {
	return c.redis.CreateCalendarEvent(ctx, event)
}

// GetCalendarEventByID retrieves a calendar event from Redis
func (c *CompositeRepository) GetCalendarEventByID(ctx context.Context, id string) (*models.CalendarEvent, error) {
	return c.redis.GetCalendarEventByID(ctx, id)
}

// UpdateCalendarEvent updates a calendar event in Redis
func (c *CompositeRepository) UpdateCalendarEvent(ctx context.Context, event *models.CalendarEvent) error {
	return c.redis.UpdateCalendarEvent(ctx, event)
}

// DeleteCalendarEvent removes a calendar event from Redis
func (c *CompositeRepository) DeleteCalendarEvent(ctx context.Context, id string) error {
	return c.redis.DeleteCalendarEvent(ctx, id)
}

// GetAllCalendarEvents retrieves all calendar events from Redis
func (c *CompositeRepository) GetAllCalendarEvents(ctx context.Context) ([]*models.CalendarEvent, error) {
	return c.redis.GetAllCalendarEvents(ctx)
}

// ===== NOTE VIEW OPERATIONS =====

// TrackNoteView records a note view in Redis
func (c *CompositeRepository) TrackNoteView(ctx context.Context, view *models.NoteView) error {
	return c.redis.TrackNoteView(ctx, view)
}

// GetViewCountByNote returns the number of views for a note
func (c *CompositeRepository) GetViewCountByNote(ctx context.Context, noteID string) (int64, error) {
	return c.redis.GetViewCountByNote(ctx, noteID)
}

// GetElasticsearchRepo returns the Elasticsearch repository instance
func (c *CompositeRepository) GetElasticsearchRepo() *ElasticsearchRepository {
	return c.es
}
