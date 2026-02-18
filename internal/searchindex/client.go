package searchindex

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/psds-microservice/session-manager-service/internal/model"
)

// SessionIndexer — интерфейс для индексации сессий в search-service (для подмены моком в тестах).
type SessionIndexer interface {
	IndexSessionAsync(ses *model.ConsultationSession)
}

// Client отправляет сессии в search-service для индексации (best-effort, не блокирует API).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient возвращает клиент. Если baseURL пустой, вызовы IndexSession — no-op.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type indexSessionPayload struct {
	SessionID string `json:"session_id"`
	ClientID  string `json:"client_id"`
	Pin       string `json:"pin"`
	Status    string `json:"status"`
}

// IndexSession отправляет сессию в search-service.
func (c *Client) IndexSession(ctx context.Context, ses *model.ConsultationSession) {
	if c.baseURL == "" || ses == nil {
		return
	}
	payload := indexSessionPayload{
		SessionID: ses.ID.String(),
		ClientID:  ses.ClientID.String(),
		Pin:       ses.PIN,
		Status:    ses.Status,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("searchindex: marshal session: %v", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search/index/session", bytes.NewReader(body))
	if err != nil {
		log.Printf("searchindex: new request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("searchindex: request session: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("searchindex: status %d for session %s", resp.StatusCode, ses.ID)
	}
}

// IndexSessionAsync вызывает IndexSession в отдельной горутине.
func (c *Client) IndexSessionAsync(ses *model.ConsultationSession) {
	if c.baseURL == "" || ses == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.IndexSession(ctx, ses)
	}()
}
