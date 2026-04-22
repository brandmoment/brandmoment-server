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

// routingRow holds the result of a single corpus-entry routing attempt.
type routingRow struct {
	phrase      string
	group       string
	primaryConf llm.ConfidenceStatus
	routedTo    llm.RoutingDecision
	finalConf   llm.ConfidenceStatus
	latencyMS   float64
	inTok       int
	outTok      int
}

// TestRoutingBenchmark runs the confidence-based routing logic against every
// corpus entry using gpt-4o-mini as primary and gpt-4o as fallback, then
// writes a summary report to finetune/confidence-check/results/report_routing.md.
//
// Usage:
//
//	OPENAI_API_KEY=sk-... go test -v -run TestRoutingBenchmark ./finetune/
func TestRoutingBenchmark(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set; skipping routing benchmark")
	}

	primary := llm.NewOpenAIClient(apiKey, "gpt-4o-mini")
	fallback := llm.NewOpenAIClient(apiKey, "gpt-4o")
	router := llm.NewRoutedParser(primary, fallback, "gpt-4o-mini", "gpt-4o")

	corpus := loadCorpus(t)
	if len(corpus) == 0 {
		t.Fatal("corpus is empty")
	}

	rows := make([]routingRow, 0, len(corpus))
	totalPrimary, totalFallback := 0, 0
	totalInTok, totalOutTok := 0, 0

	for i, entry := range corpus {
		t.Logf("[%d/%d] %s | group=%s", i+1, len(corpus), entry.Phrase, entry.Group)

		result := router.Parse(context.Background(), entry.Phrase)

		inTok := result.Primary.InputTokens
		outTok := result.Primary.OutputTokens
		latency := result.Primary.LatencyMS

		if result.Fallback != nil {
			inTok += result.Fallback.InputTokens
			outTok += result.Fallback.OutputTokens
			latency += result.Fallback.LatencyMS
		}

		totalInTok += inTok
		totalOutTok += outTok

		finalConf := result.Primary.Confidence
		if result.Fallback != nil {
			finalConf = result.Fallback.Confidence
		}

		if result.Decision == llm.RoutedPrimary {
			totalPrimary++
		} else {
			totalFallback++
		}

		rows = append(rows, routingRow{
			phrase:      entry.Phrase,
			group:       entry.Group,
			primaryConf: result.Primary.Confidence,
			routedTo:    result.Decision,
			finalConf:   finalConf,
			latencyMS:   latency,
			inTok:       inTok,
			outTok:      outTok,
		})
	}

	total := len(corpus)

	// Print results table.
	fmt.Printf("\n=== Routing Benchmark: primary=gpt-4o-mini fallback=gpt-4o corpus=%d phrases ===\n\n", total)
	fmt.Printf("%-4s | %-40s | %-7s | %-12s | %-9s | %-10s | %s\n",
		"#", "Phrase (truncated 40ch)", "Group", "Primary Conf", "Routed To", "Final Conf", "Latency")
	fmt.Printf("%s\n", strings.Repeat("-", 105))

	for i, row := range rows {
		phrase := row.phrase
		if len(phrase) > 40 {
			phrase = phrase[:37] + "..."
		}
		latencyS := fmt.Sprintf("%.1fs", row.latencyMS/1000)
		fmt.Printf("%-4d | %-40s | %-7s | %-12s | %-9s | %-10s | %s\n",
			i+1,
			phrase,
			row.group,
			string(row.primaryConf),
			string(row.routedTo),
			string(row.finalConf),
			latencyS,
		)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Stayed on primary (gpt-4o-mini): %d/%d (%.0f%%)\n",
		totalPrimary, total, 100*float64(totalPrimary)/float64(total))
	fmt.Printf("  Escalated to fallback (gpt-4o):  %d/%d (%.0f%%)\n",
		totalFallback, total, 100*float64(totalFallback)/float64(total))
	fmt.Printf("  Total input tokens:  %d\n", totalInTok)
	fmt.Printf("  Total output tokens: %d\n", totalOutTok)

	writeRoutingReportMD(t, rows, totalPrimary, totalFallback, totalInTok, totalOutTok, total)
}

func writeRoutingReportMD(
	t *testing.T,
	rows []routingRow,
	totalPrimary, totalFallback, totalInTok, totalOutTok, total int,
) {
	t.Helper()

	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filename)
	resultsDir := filepath.Join(base, "..", "..", "..", "finetune", "confidence-check", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		t.Logf("could not create results dir: %v", err)
		return
	}
	outPath := filepath.Join(resultsDir, "report_routing.md")

	var sb strings.Builder
	sb.WriteString("# Routing Benchmark Report\n\n")
	sb.WriteString("**Primary model**: gpt-4o-mini  \n")
	sb.WriteString("**Fallback model**: gpt-4o  \n")
	sb.WriteString(fmt.Sprintf("**Date**: %s  \n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Corpus**: %d phrases\n\n", total))

	sb.WriteString("## Results\n\n")
	sb.WriteString("| # | Phrase | Group | Primary Conf | Routed To | Final Conf | Latency |\n")
	sb.WriteString("|---|--------|-------|-------------|-----------|-----------|--------|\n")

	for i, row := range rows {
		phrase := row.phrase
		if len(phrase) > 40 {
			phrase = phrase[:37] + "..."
		}
		latencyS := fmt.Sprintf("%.1fs", row.latencyMS/1000)
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s | %s |\n",
			i+1,
			phrase,
			row.group,
			string(row.primaryConf),
			string(row.routedTo),
			string(row.finalConf),
			latencyS,
		))
	}

	sb.WriteString("\n## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Stayed on primary (gpt-4o-mini) | %d/%d (%.0f%%) |\n",
		totalPrimary, total, 100*float64(totalPrimary)/float64(total)))
	sb.WriteString(fmt.Sprintf("| Escalated to fallback (gpt-4o) | %d/%d (%.0f%%) |\n",
		totalFallback, total, 100*float64(totalFallback)/float64(total)))
	sb.WriteString(fmt.Sprintf("| Total input tokens | %d |\n", totalInTok))
	sb.WriteString(fmt.Sprintf("| Total output tokens | %d |\n", totalOutTok))

	if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
		t.Logf("could not write routing report: %v", err)
		return
	}
	t.Logf("routing report written to %s", outPath)
}
