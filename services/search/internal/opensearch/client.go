// Package opensearch provides a lightweight HTTP client for OpenSearch 2.x REST API.
// It uses only stdlib net/http — no opensearch-go SDK required.
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client wraps an HTTP client for the OpenSearch REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new OpenSearch client.
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IndexMapping is the JSON body for PUT /<index>.
type IndexMapping struct {
	Settings map[string]interface{} `json:"settings"`
	Mappings map[string]interface{} `json:"mappings"`
}

// EnsureIndex creates an index with the given mapping if it does not already exist.
func (c *Client) EnsureIndex(ctx context.Context, index string, mapping IndexMapping) error {
	// HEAD request to check existence
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL+"/"+index, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch: HEAD %s: %w", index, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil // already exists
	}

	// PUT to create
	body, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/"+index, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch: PUT %s: %w", index, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opensearch: create index %s: %s %s", index, resp.Status, string(b))
	}
	log.Printf("[search] created index %s", index)
	return nil
}

// IndexDoc indexes (creates or updates) a document by ID.
func (c *Client) IndexDoc(ctx context.Context, index, id string, doc interface{}) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch: index doc %s/%s: %w", index, id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opensearch: index doc %s/%s: %s %s", index, id, resp.Status, string(b))
	}
	return nil
}

// DeleteDoc removes a document from an index.
func (c *Client) DeleteDoc(ctx context.Context, index, id string) error {
	url := fmt.Sprintf("%s/%s/_doc/%s", c.baseURL, index, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch: delete doc %s/%s: %w", index, id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opensearch: delete doc %s/%s: %s %s", index, id, resp.Status, string(b))
	}
	return nil
}

// searchRequest is the OpenSearch query body.
type searchRequest struct {
	Query    map[string]interface{}   `json:"query"`
	Highlight map[string]interface{}  `json:"highlight"`
	Size     int                      `json:"size"`
}

// SearchHit is a single document hit from OpenSearch.
type SearchHit struct {
	Index  string                 `json:"_index"`
	ID     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
	Highlight map[string][]string `json:"highlight"`
}

// SearchResponse is the top-level OpenSearch response.
type SearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []SearchHit `json:"hits"`
	} `json:"hits"`
}

// Search executes a multi-match query across the given indices.
// It returns highlighted hits from all matched indices.
func (c *Client) Search(ctx context.Context, indices []string, query string, size int) (*SearchResponse, error) {
	if size <= 0 {
		size = 20
	}
	body := searchRequest{
		Query: map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"title^3", "description^2", "*"},
				"type":   "best_fields",
				"fuzziness": "AUTO",
			},
		},
		Highlight: map[string]interface{}{
			"pre_tags":  []string{"<mark>"},
			"post_tags": []string{"</mark>"},
			"fields": map[string]interface{}{
				"title":       map[string]interface{}{},
				"description": map[string]interface{}{},
			},
		},
		Size: size,
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	indexList := strings.Join(indices, ",")
	url := fmt.Sprintf("%s/%s/_search", c.baseURL, indexList)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("opensearch: search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opensearch: search: %s %s", resp.Status, string(b))
	}
	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("opensearch: decode response: %w", err)
	}
	return &result, nil
}

// Ping checks that OpenSearch is reachable.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/_cluster/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch: ping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("opensearch: ping: %s", resp.Status)
	}
	return nil
}

