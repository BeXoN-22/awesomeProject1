package tests

// TestCIVerification — намеренно провальный тест для проверки branch protection.
// Этот тест должен сделать PR красным и заблокировать merge.
func TestCIVerification(t *testing.T) {
	t.Fatal("CI check: this test is intentionally failing to verify branch protection works")
}

import (
	app "awesomeProject1/internal"
	"errors"
	"strings"
	"testing"
)

// TestCheckURLs_PrintsHeader — первая строка вывода содержит имя файла.
func TestCheckURLs_PrintsHeader(t *testing.T) {
	input := strings.NewReader("http://example.com\n")
	var out strings.Builder

	app.CheckURLs("myfile.txt", input, &out, mockChecker{code: 200})

	if !strings.Contains(out.String(), "myfile.txt") {
		t.Errorf("header missing filename, got:\n%s", out.String())
	}
}

// TestCheckURLs_200PrintsTick — статус 200 → вывод содержит ✅.
func TestCheckURLs_200PrintsTick(t *testing.T) {
	input := strings.NewReader("http://example.com\n")
	var out strings.Builder

	app.CheckURLs("f", input, &out, mockChecker{code: 200})

	if !strings.Contains(out.String(), "✅") {
		t.Errorf("expected ✅ for 200, got:\n%s", out.String())
	}
}

// TestCheckURLs_404PrintsWarning — статус != 200 → вывод содержит ⚠️.
func TestCheckURLs_404PrintsWarning(t *testing.T) {
	input := strings.NewReader("http://example.com\n")
	var out strings.Builder

	app.CheckURLs("f", input, &out, mockChecker{code: 404})

	if !strings.Contains(out.String(), "⚠️") {
		t.Errorf("expected ⚠️ for 404, got:\n%s", out.String())
	}
}

// TestCheckURLs_ErrorPrintsCross — ошибка чекера → вывод содержит ❌.
func TestCheckURLs_ErrorPrintsCross(t *testing.T) {
	input := strings.NewReader("http://bad.com\n")
	var out strings.Builder

	app.CheckURLs("f", input, &out, mockChecker{err: errors.New("refused")})

	if !strings.Contains(out.String(), "❌") {
		t.Errorf("expected ❌ for error, got:\n%s", out.String())
	}
}

// TestCheckURLs_SkipsEmptyLines — пустые строки и строки с пробелами игнорируются.
func TestCheckURLs_SkipsEmptyLines(t *testing.T) {
	input := strings.NewReader("http://a.com\n\n   \nhttp://b.com\n")
	var out strings.Builder

	app.CheckURLs("f", input, &out, mockChecker{code: 200})

	// В выводе должно быть ровно 2 URL-строки (не 4)
	count := strings.Count(out.String(), "http://")
	if count != 2 {
		t.Errorf("expected 2 URLs processed, got %d\n%s", count, out.String())
	}
}

// TestCheckURLs_MultipleURLsInOrder — несколько URL обрабатываются по порядку.
func TestCheckURLs_MultipleURLsInOrder(t *testing.T) {
	input := strings.NewReader("http://first.com\nhttp://second.com\nhttp://third.com\n")
	var out strings.Builder

	app.CheckURLs("f", input, &out, mockChecker{code: 200})

	result := out.String()
	p1 := strings.Index(result, "first")
	p2 := strings.Index(result, "second")
	p3 := strings.Index(result, "third")

	if p1 < 0 || p2 < 0 || p3 < 0 {
		t.Fatalf("not all URLs in output:\n%s", result)
	}
	if !(p1 < p2 && p2 < p3) {
		t.Errorf("wrong order: first@%d second@%d third@%d", p1, p2, p3)
	}
}

// TestCheckURLs_ReturnsNilOnSuccess — без ошибок функция возвращает nil.
func TestCheckURLs_ReturnsNilOnSuccess(t *testing.T) {
	input := strings.NewReader("http://ok.com\n")
	var out strings.Builder

	err := app.CheckURLs("f", input, &out, mockChecker{code: 200})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}