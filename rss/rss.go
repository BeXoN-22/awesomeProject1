package rss

import (
	"awesomeProject1/metrics"
	"awesomeProject1/urlcheck"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"text/tabwriter"
)

type RSSSource struct {
	ID       int
	Name     string
	URL      string
	IsActive bool
	Language string
}

func RSSSummary(source []RSSSource, checker urlcheck.Checker) error {
	return RSSSummaryTo(os.Stdout, source, checker)
}

func RSSSummaryTo(w io.Writer, source []RSSSource, checker urlcheck.Checker) error {
	buffer := tabwriter.NewWriter(w, 0, 0, 3, ' ', tabwriter.Debug)
	fmt.Fprintf(buffer, "ID\tNAME\tURL\tACTIVE\tLANGUAGE\tSTATUS\n")

	type checkResult struct {
		src    RSSSource
		status string
	}

	results := make([]checkResult, len(source))
	var wg sync.WaitGroup

	for i, src := range source {
		if !src.IsActive {
			results[i] = checkResult{src: src, status: "SKIP"}
			metrics.RSSCheckResults.WithLabelValues(src.Name, "SKIP").Inc()
			continue
		}
		wg.Add(1)
		go func(i int, src RSSSource) {
			defer wg.Done()
			res, err := checker.Check(src.URL)
			status := strconv.Itoa(res)
			if err != nil {
				status = "ERR"
			}
			results[i] = checkResult{src: src, status: status}
			metrics.RSSCheckResults.WithLabelValues(src.Name, status).Inc()
		}(i, src)
	}
	wg.Wait()

	for _, res := range results {
		fmt.Fprintf(buffer, "%d\t%s\t%s\t%t\t%s\t%s\n",
			res.src.ID, res.src.Name, res.src.URL,
			res.src.IsActive, res.src.Language, res.status)
	}
	fmt.Fprintf(buffer, "Всего проверено источников: %d\n", len(source))
	return buffer.Flush()
}
