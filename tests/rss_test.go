package tests

import (
	"awesomeProject1/rss"
	"errors"
	"strings"
	"testing"
)

// TestRSSSummaryTo_InactiveGetsSKIP — неактивный источник → строка содержит SKIP,
// чекер при этом не вызывается.
func TestRSSSummaryTo_InactiveGetsSKIP(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Inactive", URL: "http://x.com", IsActive: false, Language: "ru"},
	}
	var buf strings.Builder

	rss.RSSSummaryTo(&buf, sources, mockChecker{code: 200})

	if !strings.Contains(buf.String(), "SKIP") {
		t.Errorf("expected SKIP for inactive source, got:\n%s", buf.String())
	}
}

// TestRSSSummaryTo_ActiveGetsStatusCode — активный источник → статус-код в выводе.
func TestRSSSummaryTo_ActiveGetsStatusCode(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Active", URL: "http://x.com", IsActive: true, Language: "en"},
	}
	var buf strings.Builder

	rss.RSSSummaryTo(&buf, sources, mockChecker{code: 200})

	if !strings.Contains(buf.String(), "200") {
		t.Errorf("expected status 200 in output, got:\n%s", buf.String())
	}
}

// TestRSSSummaryTo_CheckerErrorGetsERR — ошибка чекера → строка содержит ERR.
func TestRSSSummaryTo_CheckerErrorGetsERR(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Bad", URL: "http://bad.com", IsActive: true, Language: "en"},
	}
	var buf strings.Builder

	rss.RSSSummaryTo(&buf, sources, mockChecker{err: errors.New("refused")})

	if !strings.Contains(buf.String(), "ERR") {
		t.Errorf("expected ERR for checker error, got:\n%s", buf.String())
	}
}

// TestRSSSummaryTo_OrderPreserved — порядок строк в выводе соответствует порядку источников.
// Это регрессионный тест на исправленную нами гонку.
func TestRSSSummaryTo_OrderPreserved(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "Alpha",   URL: "http://1.com", IsActive: true,  Language: "en"},
		{ID: 2, Name: "Beta",    URL: "http://2.com", IsActive: false, Language: "ru"},
		{ID: 3, Name: "Gamma",   URL: "http://3.com", IsActive: true,  Language: "en"},
		{ID: 4, Name: "Delta",   URL: "http://4.com", IsActive: false, Language: "ru"},
	}
	var buf strings.Builder

	rss.RSSSummaryTo(&buf, sources, mockChecker{code: 200})

	out := buf.String()
	positions := map[string]int{
		"Alpha": strings.Index(out, "Alpha"),
		"Beta":  strings.Index(out, "Beta"),
		"Gamma": strings.Index(out, "Gamma"),
		"Delta": strings.Index(out, "Delta"),
	}

	for name, pos := range positions {
		if pos < 0 {
			t.Fatalf("source %q not found in output", name)
		}
	}

	if !(positions["Alpha"] < positions["Beta"] &&
		positions["Beta"] < positions["Gamma"] &&
		positions["Gamma"] < positions["Delta"]) {
		t.Errorf("order wrong: %v", positions)
	}
}

// TestRSSSummaryTo_EmptyList — пустой список не вызывает паники, счётчик = 0.
func TestRSSSummaryTo_EmptyList(t *testing.T) {
	var buf strings.Builder

	err := rss.RSSSummaryTo(&buf, []rss.RSSSource{}, mockChecker{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected count 0, got:\n%s", buf.String())
	}
}

// TestRSSSummaryTo_MixedSources — активные и неактивные вместе.
func TestRSSSummaryTo_MixedSources(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "A", URL: "http://a.com", IsActive: true,  Language: "en"},
		{ID: 2, Name: "B", URL: "http://b.com", IsActive: false, Language: "ru"},
		{ID: 3, Name: "C", URL: "http://c.com", IsActive: true,  Language: "en"},
	}
	var buf strings.Builder

	rss.RSSSummaryTo(&buf, sources, mockChecker{code: 404})

	out := buf.String()
	if !strings.Contains(out, "SKIP") {
		t.Error("expected SKIP for inactive source B")
	}
	if !strings.Contains(out, "404") {
		t.Error("expected 404 for active sources")
	}
}

// TestRSSSummaryTo_CounterLine — последняя строка содержит итоговый счётчик.
func TestRSSSummaryTo_CounterLine(t *testing.T) {
	sources := []rss.RSSSource{
		{ID: 1, Name: "A", URL: "http://a.com", IsActive: true, Language: "en"},
		{ID: 2, Name: "B", URL: "http://b.com", IsActive: true, Language: "ru"},
	}
	var buf strings.Builder

	rss.RSSSummaryTo(&buf, sources, mockChecker{code: 200})

	if !strings.Contains(buf.String(), "Всего проверено источников: 2") {
		t.Errorf("expected counter line, got:\n%s", buf.String())
	}
}

// TestRSSSummaryTo_Race — запускаем с флагом -race чтобы детектор нашёл гонки.
// go test -race ./tests/...
func TestRSSSummaryTo_Race(t *testing.T) {
	sources := make([]rss.RSSSource, 20)
	for i := range sources {
		sources[i] = rss.RSSSource{
			ID: i + 1, Name: "src", URL: "http://x.com",
			IsActive: i%2 == 0, Language: "en",
		}
	}
	var buf strings.Builder
	rss.RSSSummaryTo(&buf, sources, mockChecker{code: 200})
}