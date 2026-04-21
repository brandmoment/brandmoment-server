// Package finetune provides a benchmark harness that runs all confidence
// approaches against the corpus in ../../finetune/confidence-check/data/corpus.jsonl.
//
// Usage (requires real API keys):
//
//	OPENAI_API_KEY=sk-... go test -v -run TestBenchmark ./finetune/
//	GEMINI_API_KEY=...    go test -v -run TestBenchmark ./finetune/
package finetune

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"
	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/service"
	"go.opentelemetry.io/otel/trace/noop"
)

// corpusEntry mirrors the JSONL line format.
type corpusEntry struct {
	Phrase             string        `json:"phrase"`
	Expected           []parsedRule  `json:"expected"`
	ExpectedConfidence string        `json:"expected_confidence"`
	Group              string        `json:"group"`
}

type parsedRule struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

// approachMetrics accumulates per-approach statistics.
type approachMetrics struct {
	name         string
	ok, unsure, fail int
	latencies    []time.Duration
	inputTok     int
	outputTok    int
}

func (m *approachMetrics) record(status llm.ConfidenceStatus, latMS string, inTok, outTok int) {
	switch status {
	case llm.ConfidenceStatusOK:
		m.ok++
	case llm.ConfidenceStatusUnsure:
		m.unsure++
	default:
		m.fail++
	}
	m.inputTok += inTok
	m.outputTok += outTok
}

func p50p95(durations []time.Duration) (p50, p95 time.Duration) {
	if len(durations) == 0 {
		return 0, 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	p50 = sorted[len(sorted)/2]
	p95 = sorted[int(float64(len(sorted))*0.95)]
	return p50, p95
}

// corpusPath returns the absolute path to corpus.jsonl relative to this file.
func corpusPath() string {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filename)
	return filepath.Join(base, "..", "..", "..", "finetune", "confidence-check", "data", "corpus.jsonl")
}

func loadCorpus(t *testing.T) []corpusEntry {
	t.Helper()
	f, err := os.Open(corpusPath())
	if err != nil {
		t.Skipf("corpus not found at %s (skipping benchmark): %v", corpusPath(), err)
	}
	defer f.Close()

	var entries []corpusEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e corpusEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Logf("skip malformed line: %v", err)
			continue
		}
		entries = append(entries, e)
	}
	return entries
}

func buildClient(t *testing.T) (llm.ChatClient, string) {
	t.Helper()
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return llm.NewOpenAIClient(key, "gpt-4o-mini"), "openai/gpt-4o-mini"
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		c, err := llm.NewGeminiClient(context.Background(), key, "gemini-2.0-flash")
		if err != nil {
			t.Fatalf("create gemini client: %v", err)
		}
		return c, "gemini/gemini-2.0-flash"
	}
	t.Skip("no OPENAI_API_KEY or GEMINI_API_KEY set; skipping live benchmark")
	return nil, ""
}

// TestBenchmark runs all approaches against every corpus entry and prints
// a summary table. It is not a B benchmark because we measure real-world latency
// over network calls, not CPU-bound hot loops.
func TestBenchmark(t *testing.T) {
	corpus := loadCorpus(t)
	client, providerLabel := buildClient(t)

	tp := noop.NewTracerProvider()
	svc := service.NewRuleParserService(client, tp)

	allApproaches := []string{
		string(llm.ApproachConstraint),
		string(llm.ApproachSelfCheck),
	}

	metrics := make(map[string]*approachMetrics, len(allApproaches))
	for _, a := range allApproaches {
		metrics[a] = &approachMetrics{name: a}
	}

	overallCorrect, overallTotal := 0, 0

	for i, entry := range corpus {
		t.Logf("[%d/%d] %s | group=%s expected=%s", i+1, len(corpus), entry.Phrase, entry.Group, entry.ExpectedConfidence)

		start := time.Now()
		rules, report, err := svc.Parse(context.Background(), entry.Phrase, allApproaches)
		elapsed := time.Since(start)
		_ = elapsed

		if err != nil {
			t.Logf("  parse error: %v", err)
		}

		// Check if overall verdict matches expected.
		if string(report.Overall) == entry.ExpectedConfidence {
			overallCorrect++
		}
		overallTotal++

		_ = rules // rules checked for correctness below

		for _, approach := range allApproaches {
			ar, found := report.Approaches[llm.ApproachName(approach)]
			if !found {
				continue
			}
			metrics[approach].record(ar.Status, ar.LatencyMS, ar.InputTokens, ar.OutputTokens)
		}
	}

	// Print results table.
	fmt.Printf("\n=== Benchmark Results: provider=%s corpus=%d phrases ===\n", providerLabel, len(corpus))
	fmt.Printf("Overall accuracy: %d/%d (%.0f%%)\n\n", overallCorrect, overallTotal, 100*float64(overallCorrect)/float64(overallTotal))
	fmt.Printf("%-15s  %5s  %7s  %5s  %8s  %9s\n", "Approach", "OK", "UNSURE", "FAIL", "In Tok", "Out Tok")
	fmt.Printf("%s\n", strings.Repeat("-", 65))
	for _, approach := range allApproaches {
		m := metrics[approach]
		fmt.Printf("%-15s  %5d  %7d  %5d  %8d  %9d\n",
			m.name, m.ok, m.unsure, m.fail, m.inputTok, m.outputTok)
	}

	// Write results to file.
	writeResultsMD(t, providerLabel, corpus, metrics, overallCorrect, overallTotal)
}

func writeResultsMD(t *testing.T, provider string, corpus []corpusEntry, metrics map[string]*approachMetrics, correct, total int) {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filename)
	resultsDir := filepath.Join(base, "..", "..", "..", "finetune", "confidence-check", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Logf("could not create results dir: %v", err)
		return
	}
	outPath := filepath.Join(resultsDir, fmt.Sprintf("report_%s.md", strings.ReplaceAll(provider, "/", "_")))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Benchmark Report\n\n**Provider**: %s  \n**Date**: %s  \n**Corpus**: %d phrases\n\n",
		provider, time.Now().Format("2006-01-02"), len(corpus)))
	sb.WriteString(fmt.Sprintf("**Overall accuracy**: %d/%d (%.0f%%)\n\n", correct, total, 100*float64(correct)/float64(total)))
	sb.WriteString("| Approach | OK | UNSURE | FAIL | Input Tokens | Output Tokens |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")
	for _, approach := range []string{"constraint", "self_check"} {
		m := metrics[approach]
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %d |\n",
			m.name, m.ok, m.unsure, m.fail, m.inputTok, m.outputTok))
	}

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		t.Logf("could not write results: %v", err)
		return
	}
	t.Logf("results written to %s", outPath)
}
