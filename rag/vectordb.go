package rag

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

type SearchResult struct {
	ID        string
	Score     float32
	FilePath  string
	Content   string
	LineStart int
	LineEnd   int
	Language  string
}

type CollectionInfo struct {
	PointsCount int64
	VectorDim   int
	UpdatedAt   time.Time
	Summary     string
}

type VectorDB interface {
	CreateCollection(ctx context.Context, name string, dimension int) error
	Upsert(ctx context.Context, collection string, points []Point) error
	Search(ctx context.Context, collection string, vector []float32, limit int, minScore float32) ([]SearchResult, error)
	Delete(ctx context.Context, collection string, filter map[string]interface{}) error
	GetCollectionInfo(ctx context.Context, collection string) (*CollectionInfo, error)
	Close() error
}

type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

type QdrantDB struct {
	client *qdrant.Client
}

func NewQdrantDB(host string, port int, apiKey string) (*QdrantDB, error) {
	config := &qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
	}

	// Enable TLS if API key is provided (typically for cloud)
	if apiKey != "" {
		config.UseTLS = true
	}

	client, err := qdrant.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	return &QdrantDB{
		client: client,
	}, nil
}

func (q *QdrantDB) CreateCollection(ctx context.Context, name string, dimension int) error {
	err := q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(dimension),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

func (q *QdrantDB) Upsert(ctx context.Context, collection string, points []Point) error {
	qdrantPoints := make([]*qdrant.PointStruct, len(points))

	for i, point := range points {
		// Add indexed timestamp to payload
		payload := make(map[string]interface{})
		for k, v := range point.Payload {
			payload[k] = v
		}
		payload["_indexed_at"] = time.Now().Format(time.RFC3339)

		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(point.ID),
			Vectors: qdrant.NewVectors(point.Vector...),
			Payload: qdrant.NewValueMap(payload),
		}
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         qdrantPoints,
	})

	return err
}

func (q *QdrantDB) Search(ctx context.Context, collection string, vector []float32, limit int, minScore float32) ([]SearchResult, error) {
	resp, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		ScoreThreshold: qdrant.PtrOf(minScore),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(resp))
	for i, point := range resp {
		lineStart := 0
		lineEnd := 0

		if ls := point.Payload["line_start"]; ls != nil {
			lineStart = int(ls.GetIntegerValue())
		}
		if le := point.Payload["line_end"]; le != nil {
			lineEnd = int(le.GetIntegerValue())
		}

		filePath := ""
		if fp := point.Payload["file_path"]; fp != nil {
			filePath = fp.GetStringValue()
		}

		content := ""
		if c := point.Payload["content"]; c != nil {
			content = c.GetStringValue()
		}

		language := ""
		if l := point.Payload["language"]; l != nil {
			language = l.GetStringValue()
		}

		results[i] = SearchResult{
			ID:        point.Id.GetUuid(),
			Score:     point.Score,
			FilePath:  filePath,
			Content:   content,
			Language:  language,
			LineStart: lineStart,
			LineEnd:   lineEnd,
		}
	}

	// Deduplicate results by file path and overlapping line ranges
	deduped := deduplicateResults(results)

	return deduped, nil
}

// deduplicateResults removes duplicate chunks that represent the same code
func deduplicateResults(results []SearchResult) []SearchResult {
	if len(results) == 0 {
		return results
	}

	seen := make(map[string]bool)
	unique := []SearchResult{}

	for _, result := range results {
		// Create a key based on file + line range
		// Consider chunks overlapping if they share >50% of lines
		key := fmt.Sprintf("%s:%d-%d", result.FilePath, result.LineStart, result.LineEnd)

		// Check if we've seen an overlapping chunk from this file
		isDuplicate := false
		for existingKey := range seen {
			if isOverlapping(existingKey, key) {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			seen[key] = true
			unique = append(unique, result)
		}
	}

	return unique
}

// isOverlapping checks if two file:line-range keys represent overlapping chunks
func isOverlapping(key1, key2 string) bool {
	// Parse keys: "file:start-end"
	parts1 := parseRangeKey(key1)
	parts2 := parseRangeKey(key2)

	if parts1 == nil || parts2 == nil {
		return false
	}

	// Different files = not overlapping
	if parts1[0] != parts2[0] {
		return false
	}

	// Same file, check if ranges overlap significantly
	start1, end1 := parts1[1], parts1[2]
	start2, end2 := parts2[1], parts2[2]

	// Calculate overlap
	overlapStart := max(start1, start2)
	overlapEnd := min(end1, end2)

	if overlapStart >= overlapEnd {
		return false // No overlap
	}

	overlap := overlapEnd - overlapStart
	range1 := end1 - start1
	range2 := end2 - start2

	// Consider duplicate if overlap > 50% of either range
	return float64(overlap) > float64(range1)*0.5 || float64(overlap) > float64(range2)*0.5
}

func parseRangeKey(key string) []int {
	// Parse "file:start-end"
	parts := strings.Split(key, ":")
	if len(parts) != 2 {
		return nil
	}

	rangeParts := strings.Split(parts[1], "-")
	if len(rangeParts) != 2 {
		return nil
	}

	start, err1 := strconv.Atoi(rangeParts[0])
	end, err2 := strconv.Atoi(rangeParts[1])

	if err1 != nil || err2 != nil {
		return nil
	}

	return []int{0, start, end} // 0 is placeholder for file (we compare separately)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (q *QdrantDB) Delete(ctx context.Context, collection string, filter map[string]interface{}) error {
	// Build Qdrant filter from map
	// For now, we only support filtering by file_path
	filePath, ok := filter["file_path"].(string)
	if !ok {
		return fmt.Errorf("file_path filter required")
	}

	// Delete points matching the file_path
	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collection,
		Wait:           qdrant.PtrOf(true),
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: &qdrant.Filter{
					Must: []*qdrant.Condition{
						{
							ConditionOneOf: &qdrant.Condition_Field{
								Field: &qdrant.FieldCondition{
									Key: "file_path",
									Match: &qdrant.Match{
										MatchValue: &qdrant.Match_Keyword{
											Keyword: filePath,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	return err
}

func (q *QdrantDB) GetCollectionInfo(ctx context.Context, collection string) (*CollectionInfo, error) {
	resp, err := q.client.GetCollectionInfo(ctx, collection)
	if err != nil {
		return nil, err
	}

	// Extract vector dimension from config
	vectorDim := 0
	if resp.Config != nil && resp.Config.Params != nil && resp.Config.Params.VectorsConfig != nil {
		if params := resp.Config.Params.VectorsConfig.GetParams(); params != nil {
			vectorDim = int(params.Size)
		}
	}

	pointsCount := int64(0)
	if resp.PointsCount != nil {
		pointsCount = int64(*resp.PointsCount)
	}

	info := &CollectionInfo{
		PointsCount: pointsCount,
		VectorDim:   vectorDim,
		UpdatedAt:   time.Now(),
		Summary:     fmt.Sprintf("Collection ready with %d chunks", pointsCount),
	}

	return info, nil
}

func (q *QdrantDB) Close() error {
	if q.client != nil {
		return q.client.Close()
	}
	return nil
}
