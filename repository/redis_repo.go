package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"note-stacks-backend/models"

	"github.com/redis/go-redis/v9"
)

// Repository defines the interface for all data operations
type Repository interface {
	// Note operations
	CreateNote(ctx context.Context, note *models.Note) error
	GetNoteByID(ctx context.Context, id string) (*models.Note, error)
	UpdateNote(ctx context.Context, note *models.Note) error
	DeleteNote(ctx context.Context, id string) error
	GetNotesByCover(ctx context.Context, coverID string) ([]*models.Note, error)
	SearchNotes(ctx context.Context, query string) ([]*models.Note, error)
	GetAllNotes(ctx context.Context) ([]*models.Note, error)

	// Cover operations
	CreateCover(ctx context.Context, cover *models.Cover) error
	GetCoverByID(ctx context.Context, id string) (*models.Cover, error)
	UpdateCover(ctx context.Context, cover *models.Cover) error
	DeleteCover(ctx context.Context, id string) error
	GetAllCovers(ctx context.Context) ([]*models.Cover, error)

	// Calendar Event operations
	CreateCalendarEvent(ctx context.Context, event *models.CalendarEvent) error
	GetCalendarEventByID(ctx context.Context, id string) (*models.CalendarEvent, error)
	UpdateCalendarEvent(ctx context.Context, event *models.CalendarEvent) error
	DeleteCalendarEvent(ctx context.Context, id string) error
	GetAllCalendarEvents(ctx context.Context) ([]*models.CalendarEvent, error)

	// Note View operations
	TrackNoteView(ctx context.Context, view *models.NoteView) error
	GetViewCountByNote(ctx context.Context, noteID string) (int64, error)
}

// RedisRepository implements Repository using Redis as the data store
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new instance of RedisRepository
// client: Redis client instance for database operations
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

// ===== NOTE OPERATIONS =====

// CreateNote stores a new note in Redis
// ctx: Context for the operation
// note: The note to create
func (r *RedisRepository) CreateNote(ctx context.Context, note *models.Note) error {
	noteJSON, err := json.Marshal(note)
	if err != nil {
		return fmt.Errorf("failed to marshal note: %w", err)
	}

	noteKey := fmt.Sprintf("note:%s", note.ID)
	pipe := r.client.Pipeline()

	// Store note data
	pipe.Set(ctx, noteKey, noteJSON, 0)

	// Add to cover index
	if note.Cover != "" {
		coverKey := fmt.Sprintf("cover:%s:notes", note.Cover)
		pipe.SAdd(ctx, coverKey, note.ID)
	}

	// Add to all notes index
	pipe.SAdd(ctx, "notes:all", note.ID)

	// Add to search index (lowercase title for case-insensitive search)
	pipe.SAdd(ctx, fmt.Sprintf("search:title:%s", note.Title), note.ID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	return nil
}

// GetNoteByID retrieves a note from Redis by its ID
// ctx: Context for the operation
// id: The unique identifier of the note
func (r *RedisRepository) GetNoteByID(ctx context.Context, id string) (*models.Note, error) {
	noteKey := fmt.Sprintf("note:%s", id)

	noteJSON, err := r.client.Get(ctx, noteKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("note not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	var note models.Note
	if err := json.Unmarshal([]byte(noteJSON), &note); err != nil {
		return nil, fmt.Errorf("failed to unmarshal note: %w", err)
	}

	// Populate category if Cover ID is present
	if note.Cover != "" {
		cover, err := r.GetCoverByID(ctx, note.Cover)
		if err == nil {
			note.Category = cover
		}
	}

	// 🛠️ #15.2: Populate category if Cover ID is present
	if note.Cover != "" {
		cover, err := r.GetCoverByID(ctx, note.Cover)
		if err == nil {
			// Only attach valid covers
			if cover.Valid {
				note.Category = cover
			} else {
				note.Category = nil
			}
		}
	}

	return &note, nil
}

// UpdateNote modifies an existing note in Redis
// ctx: Context for the operation
// note: The note with updated data
func (r *RedisRepository) UpdateNote(ctx context.Context, note *models.Note) error {
	oldNote, err := r.GetNoteByID(ctx, note.ID)
	if err != nil {
		return err
	}

	note.Revised = time.Now()

	noteJSON, err := json.Marshal(note)
	if err != nil {
		return fmt.Errorf("failed to marshal note: %w", err)
	}

	noteKey := fmt.Sprintf("note:%s", note.ID)
	pipe := r.client.Pipeline()

	// Update note data
	pipe.Set(ctx, noteKey, noteJSON, 0)

	// If cover changed, update cover indexes
	if oldNote.Cover != note.Cover {
		if oldNote.Cover != "" {
			oldCoverKey := fmt.Sprintf("cover:%s:notes", oldNote.Cover)
			pipe.SRem(ctx, oldCoverKey, note.ID)
		}
		if note.Cover != "" {
			newCoverKey := fmt.Sprintf("cover:%s:notes", note.Cover)
			pipe.SAdd(ctx, newCoverKey, note.ID)
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}

// DeleteNote removes a note from Redis
// ctx: Context for the operation
// id: The unique identifier of the note to delete
func (r *RedisRepository) DeleteNote(ctx context.Context, id string) error {
	note, err := r.GetNoteByID(ctx, id)
	if err != nil {
		return err
	}

	noteKey := fmt.Sprintf("note:%s", id)
	pipe := r.client.Pipeline()

	// Delete note
	pipe.Del(ctx, noteKey)

	// Remove from cover index
	if note.Cover != "" {
		coverKey := fmt.Sprintf("cover:%s:notes", note.Cover)
		pipe.SRem(ctx, coverKey, id)
	}

	// Remove from all notes index
	pipe.SRem(ctx, "notes:all", id)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

// GetNotesByCover retrieves all notes that belong to a specific cover/category
// ctx: Context for the operation
// coverID: The unique identifier of the cover
func (r *RedisRepository) GetNotesByCover(ctx context.Context, coverID string) ([]*models.Note, error) {
	coverKey := fmt.Sprintf("cover:%s:notes", coverID)

	noteIDs, err := r.client.SMembers(ctx, coverKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get note IDs: %w", err)
	}

	if len(noteIDs) == 0 {
		return []*models.Note{}, nil
	}

	notes := make([]*models.Note, 0, len(noteIDs))
	for _, noteID := range noteIDs {
		note, err := r.GetNoteByID(ctx, noteID)
		if err != nil {
			continue
		}
		if note.Display {
			notes = append(notes, note)
		}
	}

	return notes, nil
}

// SearchNotes searches for notes by title (case-insensitive substring match)
// ctx: Context for the operation
// query: The search query string
func (r *RedisRepository) SearchNotes(ctx context.Context, query string) ([]*models.Note, error) {
	allNoteIDs, err := r.client.SMembers(ctx, "notes:all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all notes: %w", err)
	}

	notes := make([]*models.Note, 0)
	for _, noteID := range allNoteIDs {
		note, err := r.GetNoteByID(ctx, noteID)
		if err != nil {
			continue
		}
		// Simple case-insensitive substring match
		if note.Display && contains(note.Title, query) {
			notes = append(notes, note)
		}
	}

	return notes, nil
}

// GetAllNotes retrieves all notes from Redis
// ctx: Context for the operation
func (r *RedisRepository) GetAllNotes(ctx context.Context) ([]*models.Note, error) {
	noteIDs, err := r.client.SMembers(ctx, "notes:all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get note IDs: %w", err)
	}

	notes := make([]*models.Note, 0, len(noteIDs))
	for _, noteID := range noteIDs {
		note, err := r.GetNoteByID(ctx, noteID)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}

	return notes, nil
}

// ===== COVER OPERATIONS =====

// CreateCover stores a new cover/category in Redis
// ctx: Context for the operation
// cover: The cover to create
func (r *RedisRepository) CreateCover(ctx context.Context, cover *models.Cover) error {
	coverJSON, err := json.Marshal(cover)
	if err != nil {
		return fmt.Errorf("failed to marshal cover: %w", err)
	}

	coverKey := fmt.Sprintf("cover:%s", cover.ID)
	pipe := r.client.Pipeline()

	pipe.Set(ctx, coverKey, coverJSON, 0)
	pipe.SAdd(ctx, "covers:all", cover.ID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cover: %w", err)
	}

	return nil
}

// GetCoverByID retrieves a cover from Redis by its ID
// ctx: Context for the operation
// id: The unique identifier of the cover
func (r *RedisRepository) GetCoverByID(ctx context.Context, id string) (*models.Cover, error) {
	coverKey := fmt.Sprintf("cover:%s", id)

	coverJSON, err := r.client.Get(ctx, coverKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("cover not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cover: %w", err)
	}

	var cover models.Cover
	if err := json.Unmarshal([]byte(coverJSON), &cover); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cover: %w", err)
	}

	return &cover, nil
}

// UpdateCover modifies an existing cover in Redis
// ctx: Context for the operation
// cover: The cover with updated data
func (r *RedisRepository) UpdateCover(ctx context.Context, cover *models.Cover) error {
	coverJSON, err := json.Marshal(cover)
	if err != nil {
		return fmt.Errorf("failed to marshal cover: %w", err)
	}

	coverKey := fmt.Sprintf("cover:%s", cover.ID)
	if err := r.client.Set(ctx, coverKey, coverJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to update cover: %w", err)
	}

	return nil
}

// DeleteCover removes a cover from Redis
// ctx: Context for the operation
// id: The unique identifier of the cover to delete
func (r *RedisRepository) DeleteCover(ctx context.Context, id string) error {
	coverKey := fmt.Sprintf("cover:%s", id)
	pipe := r.client.Pipeline()

	pipe.Del(ctx, coverKey)
	pipe.SRem(ctx, "covers:all", id)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete cover: %w", err)
	}

	return nil
}

// GetAllCovers retrieves all covers from Redis
// ctx: Context for the operation
func (r *RedisRepository) GetAllCovers(ctx context.Context) ([]*models.Cover, error) {
	coverIDs, err := r.client.SMembers(ctx, "covers:all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cover IDs: %w", err)
	}

	covers := make([]*models.Cover, 0, len(coverIDs))
	for _, coverID := range coverIDs {
		cover, err := r.GetCoverByID(ctx, coverID)
		if err != nil {
			continue
		}
		covers = append(covers, cover)
	}

	return covers, nil
}

// ===== CALENDAR EVENT OPERATIONS =====

// CreateCalendarEvent stores a new calendar event in Redis
// ctx: Context for the operation
// event: The calendar event to create
func (r *RedisRepository) CreateCalendarEvent(ctx context.Context, event *models.CalendarEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	eventKey := fmt.Sprintf("event:%s", event.ID)
	pipe := r.client.Pipeline()

	pipe.Set(ctx, eventKey, eventJSON, 0)
	pipe.SAdd(ctx, "events:all", event.ID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

// GetCalendarEventByID retrieves a calendar event from Redis by its ID
// ctx: Context for the operation
// id: The unique identifier of the event
func (r *RedisRepository) GetCalendarEventByID(ctx context.Context, id string) (*models.CalendarEvent, error) {
	eventKey := fmt.Sprintf("event:%s", id)

	eventJSON, err := r.client.Get(ctx, eventKey).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	var event models.CalendarEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &event, nil
}

// UpdateCalendarEvent modifies an existing calendar event in Redis
// ctx: Context for the operation
// event: The event with updated data
func (r *RedisRepository) UpdateCalendarEvent(ctx context.Context, event *models.CalendarEvent) error {
	event.UpdatedAt = time.Now()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	eventKey := fmt.Sprintf("event:%s", event.ID)
	if err := r.client.Set(ctx, eventKey, eventJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// DeleteCalendarEvent removes a calendar event from Redis
// ctx: Context for the operation
// id: The unique identifier of the event to delete
func (r *RedisRepository) DeleteCalendarEvent(ctx context.Context, id string) error {
	eventKey := fmt.Sprintf("event:%s", id)
	pipe := r.client.Pipeline()

	pipe.Del(ctx, eventKey)
	pipe.SRem(ctx, "events:all", id)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// GetAllCalendarEvents retrieves all calendar events from Redis
// ctx: Context for the operation
func (r *RedisRepository) GetAllCalendarEvents(ctx context.Context) ([]*models.CalendarEvent, error) {
	eventIDs, err := r.client.SMembers(ctx, "events:all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get event IDs: %w", err)
	}

	events := make([]*models.CalendarEvent, 0, len(eventIDs))
	for _, eventID := range eventIDs {
		event, err := r.GetCalendarEventByID(ctx, eventID)
		if err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// ===== NOTE VIEW OPERATIONS =====

// TrackNoteView records a note view in Redis
// ctx: Context for the operation
// view: The note view to track
func (r *RedisRepository) TrackNoteView(ctx context.Context, view *models.NoteView) error {
	viewJSON, err := json.Marshal(view)
	if err != nil {
		return fmt.Errorf("failed to marshal view: %w", err)
	}

	viewKey := fmt.Sprintf("view:%s", view.ID)
	pipe := r.client.Pipeline()

	// Store view data
	pipe.Set(ctx, viewKey, viewJSON, 0)

	// Add to note's view list
	noteViewsKey := fmt.Sprintf("note:%s:views", view.NoteID)
	pipe.SAdd(ctx, noteViewsKey, view.ID)

	// Add to all views index
	pipe.SAdd(ctx, "views:all", view.ID)

	// Add to time-sorted set for recent views
	pipe.ZAdd(ctx, "views:timeline", redis.Z{
		Score:  float64(view.ViewedAt.Unix()),
		Member: view.ID,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to track view: %w", err)
	}

	return nil
}

// GetViewCountByNote returns the number of views for a specific note
// ctx: Context for the operation
// noteID: The unique identifier of the note
func (r *RedisRepository) GetViewCountByNote(ctx context.Context, noteID string) (int64, error) {
	noteViewsKey := fmt.Sprintf("note:%s:views", noteID)
	count, err := r.client.SCard(ctx, noteViewsKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get view count: %w", err)
	}
	return count, nil
}

// Helper function for case-insensitive substring search
func contains(str, substr string) bool {
	str = toLower(str)
	substr = toLower(substr)
	return len(str) >= len(substr) && (str == substr || findSubstring(str, substr))
}

func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
