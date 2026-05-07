package yandex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type SearchClient struct {
	apiKey string
	http   *http.Client
}

func NewYandexSearchClientFromEnv() (*SearchClient, error) {
	key := strings.TrimSpace(os.Getenv("YANDEX_IAM_TOKEN"))
	if key == "" {
		return nil, errors.New("env variable YANDEX_IAM_TOKEN not found")
	}
	return &SearchClient{
		apiKey: key,
		http: &http.Client{
			Timeout: 45 * time.Second,
		},
	}, nil
}
func (c *SearchClient) requestSearch(ctx context.Context, dto SearchYandexDTO) (SearchResponse, error) {
	searchUrl := "https://searchapi.api.cloud.yandex.net/v2/web/search"

	searchDto := SearchDTO{
		Query: SearchQuery{
			SearchType: "SEARCH_TYPE_COM",
			QueryText:  dto.Query,
			FamilyMode: "FAMILY_MODE_NONE",
			Page:       dto.Page,
		},
		FolderId:       os.Getenv("YANDEX_FOLDER_ID"),
		ResponseFormat: "FORMAT_XML",
	}

	b, err := json.Marshal(searchDto)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchUrl, bytes.NewReader(b))
	if err != nil {
		return SearchResponse{}, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.http.Do(req)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("request execution error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return SearchResponse{}, fmt.Errorf("unexpected status code: %d url=%s body=%s", resp.StatusCode, searchUrl, strings.TrimSpace(string(body)))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return SearchResponse{}, fmt.Errorf("decode error: %w", err)
	}

	return searchResp, nil
}

func (c *SearchClient) Search(ctx context.Context, query string, page int) (string, error) {

	dto := SearchYandexDTO{
		Query: query,
		Page:  page,
	}

	ans, err := c.requestSearch(ctx, dto)

	if err != nil {
		return "", err
	}
	b, err := json.Marshal(ans)

	return string(b), nil
}
