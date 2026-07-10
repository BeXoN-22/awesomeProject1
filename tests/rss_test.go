package tests

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mimile-ai/mimile/rss-checker/rss"
	"github.com/mimile-ai/mimile/rss-checker/urlcheck"
)

func TestRSSSummary_InactiveGetsSKIP(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Inactive", URL: "http://x.com", IsActive: false, Language: "ru"},
	}

	results, err := rss.RSSSummary(sources, mockChecker{code: 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "SKIP" {
		t.Errorf("expected SKIP, got %q", results[0].Status)
	}
}

func TestRSSSummary_ActiveGetsStatusCode(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Active", URL: "http://x.com", IsActive: true, Language: "en"},
	}

	results, err := rss.RSSSummary(sources, mockChecker{code: 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].StatusCode != 200 {
		t.Errorf("expected status_code 200, got %d", results[0].StatusCode)
	}
	if results[0].Status != "200" {
		t.Errorf("expected status \"200\", got %q", results[0].Status)
	}
}

func TestRSSSummary_CheckerErrorGetsERR(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Bad", URL: "http://bad.com", IsActive: true, Language: "en"},
	}

	results, err := rss.RSSSummary(sources, mockChecker{err: errors.New("refused")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "ERR" {
		t.Errorf("expected ERR, got %q", results[0].Status)
	}
	if results[0].Error == "" {
		t.Error("expected non-empty error field")
	}
}

func TestRSSSummary_OrderPreserved(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Alpha", URL: "http://1.com", IsActive: true, Language: "en"},
		{ID: 2, Name: "Beta", URL: "http://2.com", IsActive: false, Language: "ru"},
		{ID: 3, Name: "Gamma", URL: "http://3.com", IsActive: true, Language: "en"},
		{ID: 4, Name: "Delta", URL: "http://4.com", IsActive: false, Language: "ru"},
	}

	results, err := rss.RSSSummary(sources, mockChecker{code: 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	names := []string{"Alpha", "Beta", "Gamma", "Delta"}
	for i, name := range names {
		if results[i].Name != name {
			t.Errorf("position %d: expected %q, got %q", i, name, results[i].Name)
		}
	}
}

func TestRSSSummary_EmptyList(t *testing.T) {
	results, err := rss.RSSSummary([]rss.RSSSource{}, mockChecker{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %d results", len(results))
	}
}

func TestRSSSummary_MixedSources(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "A", URL: "http://a.com", IsActive: true, Language: "en"},
		{ID: 2, Name: "B", URL: "http://b.com", IsActive: false, Language: "ru"},
		{ID: 3, Name: "C", URL: "http://c.com", IsActive: true, Language: "en"},
	}

	results, err := rss.RSSSummary(sources, mockChecker{code: 404})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[1].Status != "SKIP" {
		t.Errorf("expected SKIP for inactive source B, got %q", results[1].Status)
	}
	if results[0].StatusCode != 404 || results[2].StatusCode != 404 {
		t.Error("expected status_code 404 for active sources")
	}
}

func TestRSSSummary_CounterLine(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "A", URL: "http://a.com", IsActive: true, Language: "en"},
		{ID: 2, Name: "B", URL: "http://b.com", IsActive: true, Language: "ru"},
	}

	results, err := rss.RSSSummary(sources, mockChecker{code: 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRSSSummary_TimeoutGetsERR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
	}))
	defer srv.Close()

	sources := []rss.RSSSource{
		{ID: 1, Name: "Slow", URL: srv.URL, IsActive: true, Language: "en"},
	}

	results, err := rss.RSSSummary(sources, urlcheck.NewChecker())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "ERR" {
		t.Errorf("expected ERR for timeout, got %q", results[0].Status)
	}
	if results[0].Error == "" {
		t.Error("expected non-empty error field on timeout")
	}
}

func TestRSSSummary_InvalidURLGetsERR(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Bad", URL: "not-a-valid-url", IsActive: true, Language: "en"},
	}

	results, err := rss.RSSSummary(sources, urlcheck.NewChecker())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "ERR" {
		t.Errorf("expected ERR for invalid URL, got %q", results[0].Status)
	}
}

func TestRSSSummary_Race(t *testing.T) {
	sources := make([]rss.RSSSource, 20)
	for i := range sources {
		sources[i] = rss.RSSSource{
			ID: i + 1, Name: "src", URL: "http://x.com",
			IsActive: i%2 == 0, Language: "en",
		}
	}
	rss.RSSSummary(sources, mockChecker{code: 200})
}

func TestRSSSummary_StructFields(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 7, Name: "MyFeed", URL: "http://feed.com", IsActive: true, Language: "kz"},
	}

	results, err := rss.RSSSummary(sources, mockChecker{code: 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := results[0]
	if r.SourceID != 7 {
		t.Errorf("source_id: want 7, got %d", r.SourceID)
	}
	if r.Name != "MyFeed" {
		t.Errorf("name: want MyFeed, got %q", r.Name)
	}
	if r.URL != "http://feed.com" {
		t.Errorf("url: want http://feed.com, got %q", r.URL)
	}
	if r.Language != "kz" {
		t.Errorf("language: want kz, got %q", r.Language)
	}
	if r.LatencyMs < 0 {
		t.Errorf("latency_ms should be >= 0, got %d", r.LatencyMs)
	}
	if r.CheckedAt.IsZero() {
		t.Error("checked_at should not be zero")
	}
}
