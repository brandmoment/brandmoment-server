package finetune

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"
)

// microRow holds per-phrase comparison results for the micro-first benchmark.
type microRow struct {
	num              int
	phrase           string
	group            string
	expectedConf     string
	microIntent      llm.Intent
	microMargin      float64
	route            llm.Route
	baselineConf     llm.ConfidenceStatus
	twoLevelConf     llm.ConfidenceStatus
	baselineLatMS    float64
	twoLevelLatMS    float64
	baselineLLMCalls int
	twoLevelLLMCalls int
}

// TestMicroFirstBenchmark runs TwoLevelParser against all corpus phrases and
// compares results to the baseline MultiStageParser. Writes a markdown report.
//
// Usage:
//
//	OPENAI_API_KEY=sk-... go test -v -run TestMicroFirstBenchmark ./finetune/
//
// Requires prototypes.json to be built first via TestBuildPrototypes.
func TestMicroFirstBenchmark(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set; skipping micro-first benchmark")
	}

	corpus := loadCorpus(t)
	if len(corpus) == 0 {
		t.Fatal("corpus is empty")
	}

	// Load prototypes.
	prototypes := loadPrototypes(t)

	// Build clients.
	chatClient := llm.NewOpenAIClient(apiKey, "gpt-4o-mini")
	embedClient := llm.NewOpenAIEmbedClient(apiKey, "text-embedding-3-small")

	// Build parsers.
	baseline := llm.NewMultiStageParser(chatClient)
	micro := llm.NewEmbedMicro(embedClient, prototypes)
	twoLevel := llm.NewTwoLevelParser(micro, llm.NewMultiStageParser(chatClient), 0.05)

	rows := make([]microRow, 0, len(corpus))

	var (
		baselineLLMTotal   int
		twoLevelLLMTotal   int
		routeCounts        = make(map[llm.Route]int)
		baselineConfCounts = make(map[llm.ConfidenceStatus]int)
		twoLevelConfCounts = make(map[llm.ConfidenceStatus]int)
		accuracyBaseOK     int
		accuracyTwoOK      int
	)

	for i, entry := range corpus {
		t.Logf("[%d/%d] %s | group=%s expected=%s", i+1, len(corpus), entry.Phrase, entry.Group, entry.ExpectedConfidence)

		ctx := context.Background()

		// Baseline: multi-stage parse.
		baselineStart := time.Now()
		baselineResult := baseline.Parse(ctx, entry.Phrase)
		baselineLatMS := float64(time.Since(baselineStart).Milliseconds())
		baselineCalls := baselineResult.TotalCalls

		// Two-level parse.
		twoLevelStart := time.Now()
		twoResult, err := twoLevel.Parse(ctx, entry.Phrase)
		twoLevelLatMS := float64(time.Since(twoLevelStart).Milliseconds())
		if err != nil {
			t.Logf("  two-level error: %v", err)
			continue
		}

		// Estimate LLM calls for two-level result.
		twoLLMCalls := 0
		if twoResult.UsedLLM && twoResult.LLM != nil {
			twoLLMCalls = twoResult.LLM.TotalCalls
			if twoResult.Route == llm.RouteLLMWithCheck {
				twoLLMCalls += 2 // self-check adds 2 calls (parse + verify)
			}
		}

		baselineLLMTotal += baselineCalls
		twoLevelLLMTotal += twoLLMCalls
		routeCounts[twoResult.Route]++
		baselineConfCounts[baselineResult.Confidence]++
		twoLevelConfCounts[twoResult.Confidence]++

		if string(baselineResult.Confidence) == entry.ExpectedConfidence {
			accuracyBaseOK++
		}
		if string(twoResult.Confidence) == entry.ExpectedConfidence {
			accuracyTwoOK++
		}

		rows = append(rows, microRow{
			num:              i + 1,
			phrase:           entry.Phrase,
			group:            entry.Group,
			expectedConf:     entry.ExpectedConfidence,
			microIntent:      twoResult.Micro.Intent,
			microMargin:      twoResult.Micro.Margin,
			route:            twoResult.Route,
			baselineConf:     baselineResult.Confidence,
			twoLevelConf:     twoResult.Confidence,
			baselineLatMS:    baselineLatMS,
			twoLevelLatMS:    twoLevelLatMS,
			baselineLLMCalls: baselineCalls,
			twoLevelLLMCalls: twoLLMCalls,
		})
	}

	total := len(rows)
	if total == 0 {
		t.Fatal("no rows collected")
	}

	llmSavedPct := 0.0
	if baselineLLMTotal > 0 {
		llmSavedPct = 100 * float64(baselineLLMTotal-twoLevelLLMTotal) / float64(baselineLLMTotal)
	}

	// Print console summary.
	fmt.Printf("\n=== Micro-First Benchmark: corpus=%d phrases ===\n\n", total)
	fmt.Printf("LLM calls — baseline: %d, two-level: %d, saved: %.1f%%\n\n", baselineLLMTotal, twoLevelLLMTotal, llmSavedPct)
	fmt.Printf("Route distribution:\n")
	for route, count := range routeCounts {
		fmt.Printf("  %-20s %d (%.0f%%)\n", route, count, 100*float64(count)/float64(total))
	}
	fmt.Printf("\nAccuracy vs expected_confidence:\n")
	fmt.Printf("  baseline:  %d/%d (%.0f%%)\n", accuracyBaseOK, total, 100*float64(accuracyBaseOK)/float64(total))
	fmt.Printf("  two-level: %d/%d (%.0f%%)\n", accuracyTwoOK, total, 100*float64(accuracyTwoOK)/float64(total))

	writeMicroFirstReport(t, rows, total, baselineLLMTotal, twoLevelLLMTotal, llmSavedPct,
		routeCounts, baselineConfCounts, twoLevelConfCounts,
		accuracyBaseOK, accuracyTwoOK)
}

// loadPrototypes reads prototypes.json from the finetune data directory.
// Skips the test with a helpful message if the file is missing.
func loadPrototypes(t *testing.T) map[llm.Intent][]float32 {
	t.Helper()
	path := prototypesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("prototypes.json not found at %s — run TestBuildPrototypes first:\n  BUILD_PROTOTYPES=1 OPENAI_API_KEY=sk-... go test -v -run TestBuildPrototypes ./finetune/\nerror: %v", path, err)
	}

	// Unmarshal as map[string][]float32 then convert keys to llm.Intent.
	var raw map[string][]float32
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse prototypes.json: %v", err)
	}

	result := make(map[llm.Intent][]float32, len(raw))
	for k, v := range raw {
		result[llm.Intent(k)] = v
	}

	t.Logf("loaded %d prototype classes from %s", len(result), path)
	return result
}

// writeMicroFirstReport writes the per-phrase table and summary to report_microfirst.md.
func writeMicroFirstReport(
	t *testing.T,
	rows []microRow,
	total, baselineLLM, twoLevelLLM int,
	llmSavedPct float64,
	routeCounts map[llm.Route]int,
	baselineConf, twoLevelConf map[llm.ConfidenceStatus]int,
	accBase, accTwo int,
) {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filename)
	resultsDir := filepath.Join(base, "..", "..", "..", "finetune", "confidence-check", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Logf("could not create results dir: %v", err)
		return
	}
	outPath := filepath.Join(resultsDir, "report_microfirst.md")

	var sb strings.Builder
	sb.WriteString("# Micro-First Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Date**: %s  \n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Corpus**: %d phrases\n\n", total))

	sb.WriteString("## Per-Phrase Results\n\n")
	sb.WriteString("| # | Phrase | Group | Expected | Micro Intent | Margin | Route | Baseline Conf | Two-Level Conf | Baseline ms | Two-Level ms | Baseline LLM | TwoLevel LLM |\n")
	sb.WriteString("|---|--------|-------|----------|--------------|--------|-------|--------------|----------------|------------|-------------|--------------|-------------|\n")

	for _, row := range rows {
		phrase := row.phrase
		if len(phrase) > 35 {
			phrase = phrase[:32] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %.3f | %s | %s | %s | %.0f | %.0f | %d | %d |\n",
			row.num,
			phrase,
			row.group,
			row.expectedConf,
			string(row.microIntent),
			row.microMargin,
			string(row.route),
			string(row.baselineConf),
			string(row.twoLevelConf),
			row.baselineLatMS,
			row.twoLevelLatMS,
			row.baselineLLMCalls,
			row.twoLevelLLMCalls,
		))
	}

	sb.WriteString("\n## Summary\n\n")
	sb.WriteString(fmt.Sprintf("**LLM calls saved**: %d → %d (%.1f%% reduction)\n\n", baselineLLM, twoLevelLLM, llmSavedPct))

	sb.WriteString("### Route Distribution\n\n")
	sb.WriteString("| Route | Count | % |\n")
	sb.WriteString("|-------|-------|---|\n")
	for _, route := range []llm.Route{llm.RouteMicroEarlyFail, llm.RouteLLMDirect, llm.RouteLLMWithCheck} {
		count := routeCounts[route]
		sb.WriteString(fmt.Sprintf("| %s | %d | %.0f%% |\n", route, count, 100*float64(count)/float64(total)))
	}

	sb.WriteString("\n### Confidence Distribution\n\n")
	sb.WriteString("| Confidence | Baseline | Two-Level |\n")
	sb.WriteString("|------------|----------|----------|\n")
	for _, status := range []llm.ConfidenceStatus{llm.ConfidenceStatusOK, llm.ConfidenceStatusUnsure, llm.ConfidenceStatusFail} {
		sb.WriteString(fmt.Sprintf("| %s | %d | %d |\n", status, baselineConf[status], twoLevelConf[status]))
	}

	sb.WriteString("\n### Accuracy vs expected_confidence\n\n")
	sb.WriteString("| Parser | Correct | Total | Accuracy |\n")
	sb.WriteString("|--------|---------|-------|----------|\n")
	sb.WriteString(fmt.Sprintf("| Baseline (MultiStage) | %d | %d | %.0f%% |\n",
		accBase, total, 100*float64(accBase)/float64(total)))
	sb.WriteString(fmt.Sprintf("| Two-Level (MicroFirst) | %d | %d | %.0f%% |\n",
		accTwo, total, 100*float64(accTwo)/float64(total)))

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		t.Logf("could not write microfirst report: %v", err)
		return
	}
	t.Logf("microfirst report written to %s", outPath)
}
