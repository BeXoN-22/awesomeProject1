package app

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mimile-ai/mimile/rss-checker/urlcheck"
)

func Run(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("открытая ссылка не открывается: %w", err)
	}
	defer file.Close()

	return CheckURLs(filename, file, os.Stdout, urlcheck.NewChecker())
}

func CheckURLs(filename string, reader io.Reader, writer io.Writer, checker urlcheck.Checker) error {
	fmt.Fprintf(writer, "--- Проверка сайтов из файла %s ---\n", filename)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url == "" {
			continue
		}
		statusCode, err := checker.Check(url)
		printResult(writer, url, statusCode, err)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка при чтении файла :%w", err)
	}
	return nil
}

func printResult(writer io.Writer, url string, statusCode int, err error) {
	if err != nil {
		fmt.Fprintf(writer, "❌ %s — Ошибка: %v\n", url, err)
		return
	}

	if statusCode == http.StatusOK {
		fmt.Fprintf(writer, "✅ %s — Статус: %d OK\n", url, statusCode)
		return
	}

	fmt.Fprintf(writer, "⚠️ %s — Статус: %d\n", url, statusCode)
}
