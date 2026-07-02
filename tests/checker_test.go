package tests

import (
	"awesomeProject1/urlcheck"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestCheck_StatusCodes — table-driven: сервер отвечает разными кодами,
// HTTPChecker должен вернуть тот же код без ошибки.
func TestCheck_StatusCodes(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"200 OK", 200},
		{"301 Redirect", 301},
		{"403 Forbidden", 403},
		{"404 Not Found", 404},
		{"500 Server Error", 500},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			code, err := urlcheck.NewChecker().Check(srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if code != tc.status {
				t.Errorf("got %d, want %d", code, tc.status)
			}
		})
	}
}

// TestCheck_UsesHEADMethod — проверяем что запрос отправляется методом HEAD, а не GET.
func TestCheck_UsesHEADMethod(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(200)
	}))
	defer srv.Close()

	urlcheck.NewChecker().Check(srv.URL)

	if gotMethod != http.MethodHead {
		t.Errorf("want HEAD, got %s", gotMethod)
	}
}

// TestCheck_SetsUserAgent — проверяем что заголовок User-Agent не пустой и содержит "Mozilla".
func TestCheck_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	urlcheck.NewChecker().Check(srv.URL)

	if gotUA == "" {
		t.Error("User-Agent is empty")
	}
	if !strings.Contains(gotUA, "Mozilla") {
		t.Errorf("expected Mozilla in User-Agent, got: %s", gotUA)
	}
}

// TestCheck_NetworkError — сервер закрыт до запроса, ожидаем ошибку сети.
func TestCheck_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // закрываем ДО вызова Check

	_, err := urlcheck.NewChecker().Check(url)
	if err == nil {
		t.Error("expected network error, got nil")
	}
}

// TestCheck_Timeout — сервер зависает дольше таймаута клиента (3s), ожидаем ошибку.
func TestCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
	}))
	defer srv.Close()

	_, err := urlcheck.NewChecker().Check(srv.URL)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

// TestCheck_InvalidURL — невалидный URL, ожидаем ошибку без паники.
func TestCheck_InvalidURL(t *testing.T) {
	_, err := urlcheck.NewChecker().Check("not-a-valid-url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

// TestCheck_EmptyURL — пустая строка URL.
func TestCheck_EmptyURL(t *testing.T) {
	_, err := urlcheck.NewChecker().Check("")
	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
}