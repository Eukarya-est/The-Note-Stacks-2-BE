package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"note-stacks-backend/models"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

const (
	notesIndex = "notes"
)

// ElasticsearchRepository implements search functionality using Elasticsearch
type ElasticsearchRepository struct {
	client *elasticsearch.Client
	repo   Repository // Underlying Redis repository for data operations
}

// NewElasticsearchRepository creates a new Elasticsearch repository
func NewElasticsearchRepository(esClient *elasticsearch.Client, redisRepo Repository) *ElasticsearchRepository {
	return &ElasticsearchRepository{
		client: esClient,
		repo:   redisRepo,
	}
}

// IndexNote indexes a note in Elasticsearch for searching
func (e *ElasticsearchRepository) IndexNote(ctx context.Context, note *models.Note) error {
	// Prepare the document for indexing
	doc := map[string]interface{}{
		"id":       note.ID,
		"title":    note.Title,
		"content":  note.Content,
		"filename": note.Filename,
		"filepath": note.Filepath,
		"cover":    note.Cover,
		"revision": note.Revision,
		"created":  note.Created,
		"revised":  note.Revised,
		"display":  note.Display,
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	// Index the document
	req := esapi.IndexRequest{
		Index:      notesIndex,
		DocumentID: note.ID,
		Body:       bytes.NewReader(docJSON),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch error: %s", res.String())
	}

	return nil
}

// DeleteNoteFromIndex removes a note from Elasticsearch index
func (e *ElasticsearchRepository) DeleteNoteFromIndex(ctx context.Context, noteID string) error {
	req := esapi.DeleteRequest{
		Index:      notesIndex,
		DocumentID: noteID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("elasticsearch error: %s", res.String())
	}

	return nil
}

// SearchNotes performs a full-text search on notes using Elasticsearch
func (e *ElasticsearchRepository) SearchNotes(ctx context.Context, query string) ([]*models.Note, error) {
	// Build the search query
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":     query,
							"fields":    []string{"title^3", "content", "filename^2"},
							"type":      "best_fields",
							"fuzziness": "AUTO",
						},
					},
				},
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"display": true,
						},
					},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"title":   map[string]interface{}{},
				"content": map[string]interface{}{},
			},
		},
		"size": 100,
	}

	queryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Perform the search
	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(notesIndex),
		e.client.Search.WithBody(bytes.NewReader(queryJSON)),
		e.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract note IDs from hits
	hits, ok := response["hits"].(map[string]interface{})
	if !ok {
		return []*models.Note{}, nil
	}

	hitsList, ok := hits["hits"].([]interface{})
	if !ok {
		return []*models.Note{}, nil
	}

	// Fetch full note details from Redis
	noteIDs := make([]string, 0, len(hitsList))
	for _, hit := range hitsList {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		if id, ok := source["id"].(string); ok {
			noteIDs = append(noteIDs, id)
		}
	}

	// Fetch notes from Redis repository
	notes := make([]*models.Note, 0, len(noteIDs))
	for _, noteID := range noteIDs {
		note, err := e.repo.GetNoteByID(ctx, noteID)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}

	return notes, nil
}

// AdvancedSearch performs advanced search with filters
func (e *ElasticsearchRepository) AdvancedSearch(ctx context.Context, params SearchParams) ([]*models.Note, error) {
	mustClauses := []interface{}{}
	filterClauses := []interface{}{
		map[string]interface{}{
			"term": map[string]interface{}{
				"display": true,
			},
		},
	}

	// Add text search if query is provided
	if params.Query != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     params.Query,
				"fields":    []string{"title^3", "content", "filename^2"},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		})
	}

	// Add cover filter if specified
	if params.CoverID != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"cover": params.CoverID,
			},
		})
	}

	// Add date range filter if specified
	if params.DateFrom != "" || params.DateTo != "" {
		rangeQuery := map[string]interface{}{}
		if params.DateFrom != "" {
			rangeQuery["gte"] = params.DateFrom
		}
		if params.DateTo != "" {
			rangeQuery["lte"] = params.DateTo
		}

		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{
				"created": rangeQuery,
			},
		})
	}

	// Build the complete query
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   mustClauses,
				"filter": filterClauses,
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"title":   map[string]interface{}{},
				"content": map[string]interface{}{},
			},
		},
		"size": params.Size,
		"from": params.From,
	}

	// Add sorting if specified
	if params.SortBy != "" {
		order := "desc"
		if params.SortOrder != "" {
			order = params.SortOrder
		}

		searchQuery["sort"] = []interface{}{
			map[string]interface{}{
				params.SortBy: map[string]interface{}{
					"order": order,
				},
			},
		}
	}

	queryJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Perform the search
	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(notesIndex),
		e.client.Search.WithBody(bytes.NewReader(queryJSON)),
		e.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	// Parse and return results
	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	hits, ok := response["hits"].(map[string]interface{})
	if !ok {
		return []*models.Note{}, nil
	}

	hitsList, ok := hits["hits"].([]interface{})
	if !ok {
		return []*models.Note{}, nil
	}

	noteIDs := make([]string, 0, len(hitsList))
	for _, hit := range hitsList {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		if id, ok := source["id"].(string); ok {
			noteIDs = append(noteIDs, id)
		}
	}

	notes := make([]*models.Note, 0, len(noteIDs))
	for _, noteID := range noteIDs {
		note, err := e.repo.GetNoteByID(ctx, noteID)
		if err != nil {
			continue
		}
		notes = append(notes, note)
	}

	return notes, nil
}

// SearchParams defines parameters for advanced search
type SearchParams struct {
	Query     string
	CoverID   string
	DateFrom  string
	DateTo    string
	SortBy    string
	SortOrder string
	Size      int
	From      int
}

// InitializeIndex creates the notes index with proper mappings
func (e *ElasticsearchRepository) InitializeIndex(ctx context.Context) error {
	// Check if index exists
	res, err := e.client.Indices.Exists([]string{notesIndex})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	// If index exists, return
	if res.StatusCode == 200 {
		return nil
	}

	// Create index with mappings
	mapping := `{
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"title": { 
					"type": "text",
					"analyzer": "standard",
					"fields": {
						"keyword": { "type": "keyword" }
					}
				},
				"content": { 
					"type": "text",
					"analyzer": "standard"
				},
				"filename": {
					"type": "text",
					"fields": {
						"keyword": { "type": "keyword" }
					}
				},
				"filepath": { "type": "keyword" },
				"cover": { "type": "keyword" },
				"revision": { "type": "keyword" },
				"created": { "type": "date" },
				"revised": { "type": "date" },
				"display": { "type": "boolean" }
			}
		},
		"settings": {
			"number_of_shards": 1,
			"number_of_replicas": 1,
			"analysis": {
				"analyzer": {
					"standard": {
						"type": "standard"
					}
				}
			}
		}
	}`

	createRes, err := e.client.Indices.Create(
		notesIndex,
		e.client.Indices.Create.WithBody(strings.NewReader(mapping)),
		e.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("failed to create index: %s", createRes.String())
	}

	return nil
}

// ReindexAllNotes reindexes all notes from Redis to Elasticsearch
func (e *ElasticsearchRepository) ReindexAllNotes(ctx context.Context) error {
	// Get all notes from Redis
	notes, err := e.repo.GetAllNotes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all notes: %w", err)
	}

	// Index each note
	for _, note := range notes {
		if err := e.IndexNote(ctx, note); err != nil {
			// Log error but continue with other notes
			fmt.Printf("Failed to index note %s: %v\n", note.ID, err)
		}
	}

	return nil
}
