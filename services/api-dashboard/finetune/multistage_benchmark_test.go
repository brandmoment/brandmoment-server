package finetune

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"
)

// monoResult holds the outcome of a single monolithic parse call.
type monoResult struct {
	rulesJSON  string
	confidence llm.ConfidenceStatus
	latencyMS  float64
	inTok      int
	outTok     int
}

// runMonolithic performs a single-call parse and constraint check.
func runMonolithic(ctx context.Context, client llm.ChatClient, phrase string) monoResult {
	start := time.Now()
	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: llm.SystemPrompt},
			{Role: llm.RoleUser, Content: phrase},
		},
		Temperature: 0,
	}
	resp, err := client.Complete(ctx, req)
	latencyMS := float64(time.Since(start).Milliseconds())

	if err != nil {
		return monoResult{
			rulesJSON:  "[]",
			confidence: llm.ConfidenceStatusFail,
			latencyMS:  latencyMS,
			inTok:      resp.InputTokens,
			outTok:     resp.OutputTokens,
		}
	}

	raw := llm.StripMarkdownFences(resp.Content)
	constraint := llm.CheckConstraint(ctx, raw)

	return monoResult{
		rulesJSON:  raw,
		confidence: constraint.Status,
		latencyMS:  latencyMS,
		inTok:      resp.InputTokens,
		outTok:     resp.OutputTokens,
	}
}

// multistageRow holds per-phrase comparison results.
type multistageRow struct {
	num        int
	phrase     string
	group      string
	monoConf   llm.ConfidenceStatus
	multiConf  llm.ConfidenceStatus
	monoMS     float64
	multiMS    float64
	monoTok    int
	multiTok   int
}

// TestMultiStageBenchmark compares the monolithic single-call approach against
// the multi-stage (3-call) approach on the shared corpus.
//
// Usage:
//
//	OPENAI_API_KEY=sk-... go test -v -run TestMultiStageBenchmark ./finetune/
func TestMultiStageBenchmark(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set; skipping multi-stage benchmark")
	}

	const model = "gpt-4o-mini"
	client := llm.NewOpenAIClient(apiKey, model)
	parser := llm.NewMultiStageParser(client)

	corpus := loadCorpus(t)
	if len(corpus) == 0 {
		t.Fatal("corpus is empty")
	}

	rows := make([]multistageRow, 0, len(corpus))

	// Per-approach accumulators.
	var (
		monoOK, monoUnsure, monoFail     int
		multiOK, multiUnsure, multiFail  int
		monoTotalMS, multiTotalMS        float64
		monoTotalTok, multiTotalTok      int
	)

	for i, entry := range corpus {
		t.Logf("[%d/%d] %s | group=%s", i+1, len(corpus), entry.Phrase, entry.Group)

		ctx := context.Background()

		mono := runMonolithic(ctx, client, entry.Phrase)
		multi := parser.Parse(ctx, entry.Phrase)

		monoTok := mono.inTok + mono.outTok
		multiTok := multi.TotalIn + multi.TotalOut

		switch mono.confidence {
		case llm.ConfidenceStatusOK:
			monoOK++
		case llm.ConfidenceStatusUnsure:
			monoUnsure++
		default:
			monoFail++
		}
		switch multi.Confidence {
		case llm.ConfidenceStatusOK:
			multiOK++
		case llm.ConfidenceStatusUnsure:
			multiUnsure++
		default:
			multiFail++
		}

		monoTotalMS += mono.latencyMS
		multiTotalMS += multi.TotalLatency
		monoTotalTok += monoTok
		multiTotalTok += multiTok

		rows = append(rows, multistageRow{
			num:       i + 1,
			phrase:    entry.Phrase,
			group:     entry.Group,
			monoConf:  mono.confidence,
			multiConf: multi.Confidence,
			monoMS:    mono.latencyMS,
			multiMS:   multi.TotalLatency,
			monoTok:   monoTok,
			multiTok:  multiTok,
		})
	}

	total := len(corpus)

	// Count wins.
	multiWins, monoWins := 0, 0
	for _, row := range rows {
		monoScore := confidenceScore(row.monoConf)
		multiScore := confidenceScore(row.multiConf)
		if multiScore > monoScore {
			multiWins++
		} else if monoScore > multiScore {
			monoWins++
		}
	}

	// Print table header.
	fmt.Printf("\n=== Multi-Stage vs Monolithic Benchmark: model=%s corpus=%d phrases ===\n\n", model, total)
	fmt.Printf("%-4s | %-40s | %-7s | %-10s | %-10s | %-8s | %-8s | %-9s | %-9s\n",
		"#", "Phrase (40ch)", "Group", "Mono Conf", "Multi Conf", "Mono ms", "Multi ms", "Mono Tok", "Multi Tok")
	fmt.Printf("%s\n", strings.Repeat("-", 115))

	for _, row := range rows {
		phrase := row.phrase
		if len(phrase) > 40 {
			phrase = phrase[:37] + "..."
		}
		fmt.Printf("%-4d | %-40s | %-7s | %-10s | %-10s | %-8.0f | %-8.0f | %-9d | %-9d\n",
			row.num,
			phrase,
			row.group,
			string(row.monoConf),
			string(row.multiConf),
			row.monoMS,
			row.multiMS,
			row.monoTok,
			row.multiTok,
		)
	}

	// Print summary.
	avgMonoS := monoTotalMS / float64(total) / 1000
	avgMultiS := multiTotalMS / float64(total) / 1000
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Monolithic:  OK=%d UNSURE=%d FAIL=%d | avg_latency=%.2fs | total_tokens=%d\n",
		monoOK, monoUnsure, monoFail, avgMonoS, monoTotalTok)
	fmt.Printf("  Multi-stage: OK=%d UNSURE=%d FAIL=%d | avg_latency=%.2fs | total_tokens=%d\n",
		multiOK, multiUnsure, multiFail, avgMultiS, multiTotalTok)
	fmt.Printf("  Multi-stage wins on confidence: %d phrases\n", multiWins)
	fmt.Printf("  Monolithic wins on confidence:  %d phrases\n", monoWins)

	writeMultistageReportMD(t, model, rows, total,
		monoOK, monoUnsure, monoFail,
		multiOK, multiUnsure, multiFail,
		avgMonoS, avgMultiS,
		monoTotalTok, multiTotalTok,
		multiWins, monoWins,
	)
}

// confidenceScore converts a ConfidenceStatus to a numeric score for comparison.
func confidenceScore(s llm.ConfidenceStatus) int {
	switch s {
	case llm.ConfidenceStatusOK:
		return 2
	case llm.ConfidenceStatusUnsure:
		return 1
	default:
		return 0
	}
}

func writeMultistageReportMD(
	t *testing.T,
	model string,
	rows []multistageRow,
	total int,
	monoOK, monoUnsure, monoFail int,
	multiOK, multiUnsure, multiFail int,
	avgMonoS, avgMultiS float64,
	monoTotalTok, multiTotalTok int,
	multiWins, monoWins int,
) {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filename)
	resultsDir := filepath.Join(base, "..", "..", "..", "finetune", "confidence-check", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Logf("could not create results dir: %v", err)
		return
	}
	outPath := filepath.Join(resultsDir, "report_multistage.md")

	var sb strings.Builder
	sb.WriteString("# Multi-Stage vs Monolithic Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Model**: %s  \n", model))
	sb.WriteString(fmt.Sprintf("**Date**: %s  \n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Corpus**: %d phrases\n\n", total))

	sb.WriteString("## Per-Phrase Results\n\n")
	sb.WriteString("| # | Phrase | Group | Mono Conf | Multi Conf | Mono ms | Multi ms | Mono Tok | Multi Tok |\n")
	sb.WriteString("|---|--------|-------|-----------|-----------|---------|---------|---------|----------|\n")

	for _, row := range rows {
		phrase := row.phrase
		if len(phrase) > 40 {
			phrase = phrase[:37] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %.0f | %.0f | %d | %d |\n",
			row.num,
			phrase,
			row.group,
			string(row.monoConf),
			string(row.multiConf),
			row.monoMS,
			row.multiMS,
			row.monoTok,
			row.multiTok,
		))
	}

	sb.WriteString("\n## Summary\n\n")
	sb.WriteString("| Metric | Monolithic | Multi-Stage |\n")
	sb.WriteString("|--------|-----------|-------------|\n")
	sb.WriteString(fmt.Sprintf("| OK | %d | %d |\n", monoOK, multiOK))
	sb.WriteString(fmt.Sprintf("| UNSURE | %d | %d |\n", monoUnsure, multiUnsure))
	sb.WriteString(fmt.Sprintf("| FAIL | %d | %d |\n", monoFail, multiFail))
	sb.WriteString(fmt.Sprintf("| Avg latency | %.2fs | %.2fs |\n", avgMonoS, avgMultiS))
	sb.WriteString(fmt.Sprintf("| Total tokens | %d | %d |\n", monoTotalTok, multiTotalTok))
	sb.WriteString(fmt.Sprintf("\n**Multi-stage wins on confidence**: %d phrases  \n", multiWins))
	sb.WriteString(fmt.Sprintf("**Monolithic wins on confidence**: %d phrases\n", monoWins))

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		t.Logf("could not write multistage report: %v", err)
		return
	}
	t.Logf("multistage report written to %s", outPath)
}
