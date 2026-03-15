package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"note-stacks-backend/models"

	"github.com/redis/go-redis/v9"
)

// CalendarRepository extends Repository with calendar-specific operations
type CalendarRepository interface {
	Repository

	// Enhanced calendar operations
	GetEventsByDateRange(ctx context.Context, startDate, endDate string) ([]*models.CalendarEvent, error)
	GetEventsByLabel(ctx context.Context, label string) ([]*models.CalendarEvent, error)
	GetEventsByMonth(ctx context.Context, year, month int) ([]*models.CalendarEvent, error)
	GetEventsByDate(ctx context.Context, date string) ([]*models.CalendarEvent, error)
	BulkCreateEvents(ctx context.Context, events []*models.CalendarEvent) error
	SearchEvents(ctx context.Context, query string) ([]*models.CalendarEvent, error)
}

// GetEventsByDateRange retrieves events within a specific date range
// startDate and endDate should be in YYYY-MM-DD format
func (r *RedisRepository) GetEventsByDateRange(ctx context.Context, startDate, endDate string) ([]*models.CalendarEvent, error) {
	// Get all events
	allEvents, err := r.GetAllCalendarEvents(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by date range
	filtered := make([]*models.CalendarEvent, 0)
	for _, event := range allEvents {
		if event.EventDate >= startDate && event.EventDate <= endDate {
			filtered = append(filtered, event)
		}
	}

	// Sort by date ascending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].EventDate < filtered[j].EventDate
	})

	return filtered, nil
}

// GetEventsByLabel retrieves all events with a specific label
func (r *RedisRepository) GetEventsByLabel(ctx context.Context, label string) ([]*models.CalendarEvent, error) {
	// Get all events
	allEvents, err := r.GetAllCalendarEvents(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by label
	filtered := make([]*models.CalendarEvent, 0)
	for _, event := range allEvents {
		if event.Label == label {
			filtered = append(filtered, event)
		}
	}

	// Sort by date ascending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].EventDate < filtered[j].EventDate
	})

	return filtered, nil
}

// GetEventsByMonth retrieves all events for a specific month
func (r *RedisRepository) GetEventsByMonth(ctx context.Context, year, month int) ([]*models.CalendarEvent, error) {
	// Create start and end dates for the month
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	return r.GetEventsByDateRange(ctx, startDateStr, endDateStr)
}

// GetEventsByDate retrieves all events for a specific date
func (r *RedisRepository) GetEventsByDate(ctx context.Context, date string) ([]*models.CalendarEvent, error) {
	return r.GetEventsByDateRange(ctx, date, date)
}

// BulkCreateEvents creates multiple events in a single transaction
func (r *RedisRepository) BulkCreateEvents(ctx context.Context, events []*models.CalendarEvent) error {
	if len(events) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	for _, event := range events {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", event.ID, err)
		}

		eventKey := fmt.Sprintf("event:%s", event.ID)
		pipe.Set(ctx, eventKey, eventJSON, 0)
		pipe.SAdd(ctx, "events:all", event.ID)

		// Add to date index for faster date-based queries
		dateKey := fmt.Sprintf("events:date:%s", event.EventDate)
		pipe.SAdd(ctx, dateKey, event.ID)

		// Add to label index for faster label-based queries
		labelKey := fmt.Sprintf("events:label:%s", event.Label)
		pipe.SAdd(ctx, labelKey, event.ID)

		// Add to month index for faster month-based queries
		if len(event.EventDate) >= 7 {
			monthKey := fmt.Sprintf("events:month:%s", event.EventDate[:7])
			pipe.SAdd(ctx, monthKey, event.ID)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to bulk create events: %w", err)
	}

	return nil
}

// SearchEvents searches for events by title or description
func (r *RedisRepository) SearchEvents(ctx context.Context, query string) ([]*models.CalendarEvent, error) {
	// Get all events
	allEvents, err := r.GetAllCalendarEvents(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by search query (case-insensitive)
	filtered := make([]*models.CalendarEvent, 0)
	queryLower := toLower(query)

	for _, event := range allEvents {
		titleLower := toLower(event.Title)
		descLower := toLower(event.Description)

		if contains(titleLower, queryLower) || contains(descLower, queryLower) {
			filtered = append(filtered, event)
		}
	}

	// Sort by date ascending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].EventDate < filtered[j].EventDate
	})

	return filtered, nil
}

// UpdateIndexesForEvent updates all secondary indexes when an event is modified
func (r *RedisRepository) UpdateIndexesForEvent(ctx context.Context, oldEvent, newEvent *models.CalendarEvent) error {
	pipe := r.client.Pipeline()

	// Remove old indexes
	if oldEvent != nil {
		// Remove from old date index
		oldDateKey := fmt.Sprintf("events:date:%s", oldEvent.EventDate)
		pipe.SRem(ctx, oldDateKey, oldEvent.ID)

		// Remove from old label index
		oldLabelKey := fmt.Sprintf("events:label:%s", oldEvent.Label)
		pipe.SRem(ctx, oldLabelKey, oldEvent.ID)

		// Remove from old month index
		if len(oldEvent.EventDate) >= 7 {
			oldMonthKey := fmt.Sprintf("events:month:%s", oldEvent.EventDate[:7])
			pipe.SRem(ctx, oldMonthKey, oldEvent.ID)
		}
	}

	// Add new indexes
	if newEvent != nil {
		// Add to new date index
		newDateKey := fmt.Sprintf("events:date:%s", newEvent.EventDate)
		pipe.SAdd(ctx, newDateKey, newEvent.ID)

		// Add to new label index
		newLabelKey := fmt.Sprintf("events:label:%s", newEvent.Label)
		pipe.SAdd(ctx, newLabelKey, newEvent.ID)

		// Add to new month index
		if len(newEvent.EventDate) >= 7 {
			newMonthKey := fmt.Sprintf("events:month:%s", newEvent.EventDate[:7])
			pipe.SAdd(ctx, newMonthKey, newEvent.ID)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

// GetEventCountByLabel returns the count of events for each label
func (r *RedisRepository) GetEventCountByLabel(ctx context.Context) (map[string]int64, error) {
	labels := []string{"reading", "repeat", "shadowing", "solve", "writing", "development"}
	counts := make(map[string]int64)

	for _, label := range labels {
		labelKey := fmt.Sprintf("events:label:%s", label)
		count, err := r.client.SCard(ctx, labelKey).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("failed to count events for label %s: %w", label, err)
		}
		counts[label] = count
	}

	return counts, nil
}

// GetEventCountByMonth returns the count of events for each month
func (r *RedisRepository) GetEventCountByMonth(ctx context.Context) (map[string]int64, error) {
	// Get all events to determine which months have events
	allEvents, err := r.GetAllCalendarEvents(ctx)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	months := make(map[string]bool)

	// Collect unique months
	for _, event := range allEvents {
		if len(event.EventDate) >= 7 {
			month := event.EventDate[:7]
			months[month] = true
		}
	}

	// Count events for each month
	for month := range months {
		monthKey := fmt.Sprintf("events:month:%s", month)
		count, err := r.client.SCard(ctx, monthKey).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("failed to count events for month %s: %w", month, err)
		}
		counts[month] = count
	}

	return counts, nil
}
