package githubapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestGetAllRequestsTenThousandRecordsInOneHundredCalls(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	const (
		pageCount = 100
		pageSize  = 100
	)
	client := &mockRESTClient{
		requests:  make(map[string]int),
		responses: make(map[string]*http.Response, pageCount),
	}
	for page := 1; page <= pageCount; page++ {
		url := fmt.Sprintf("items?per_page=100&page=%d", page)
		pageItems := make([]item, pageSize)
		for index := range pageItems {
			pageItems[index].Name = fmt.Sprintf("item-%d", (page-1)*pageSize+index)
		}
		body, _ := json.Marshal(pageItems)
		next := ""
		if page < pageCount {
			nextURL := fmt.Sprintf("items?per_page=100&page=%d", page+1)
			lastURL := fmt.Sprintf("items?per_page=100&page=%d", pageCount)
			next = fmt.Sprintf("<%s>; rel=\"next\", <%s>; rel=\"last\"", nextURL, lastURL)
		}
		client.responses[url] = jsonResponse(string(body), next)
	}

	items, err := GetAll[item](client, "items?per_page=100&page=1")
	if err != nil {
		t.Fatalf("GetAll returned error: %v", err)
	}
	if len(items) != pageCount*pageSize {
		t.Fatalf("expected %d items, got %d", pageCount*pageSize, len(items))
	}
	totalRequests := 0
	for _, count := range client.requests {
		totalRequests += count
	}
	if totalRequests != pageCount {
		t.Fatalf("expected %d requests, got %d", pageCount, totalRequests)
	}
}
