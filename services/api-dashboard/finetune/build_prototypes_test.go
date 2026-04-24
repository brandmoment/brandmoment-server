package finetune

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/brandmoment/brandmoment-server/services/api-dashboard/internal/llm"
)

// TestBuildPrototypes embeds each corpus phrase and writes mean embeddings per
// intent class to prototypes.json. It is an offline utility that must be run
// once to seed the prototypes file before benchmarks can use it.
//
// Usage:
//
//	OPENAI_API_KEY=sk-... BUILD_PROTOTYPES=1 go test -v -run TestBuildPrototypes ./finetune/
func TestBuildPrototypes(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set; skipping prototype build")
	}
	if os.Getenv("BUILD_PROTOTYPES") != "1" {
		t.Skip("BUILD_PROTOTYPES=1 not set; skipping prototype build (run intentionally to regenerate)")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	embedClient := llm.NewOpenAIEmbedClient(apiKey, "text-embedding-3-small")

	corpus := loadCorpus(t)
	if len(corpus) == 0 {
		t.Fatal("corpus is empty")
	}

	// Map from group → intent label.
	groupToIntent := map[string]llm.Intent{
		"correct": llm.IntentValid,
		"edge":    llm.IntentAmbiguous,
		"noisy":   llm.IntentGibberish,
	}

	// Accumulate embeddings per intent.
	byIntent := make(map[llm.Intent][][]float32)

	for i, entry := range corpus {
		intent, ok := groupToIntent[entry.Group]
		if !ok {
			t.Logf("[%d] skip unknown group %q for phrase %q", i+1, entry.Group, entry.Phrase)
			continue
		}

		t.Logf("[%d/%d] embed phrase=%q group=%s intent=%s", i+1, len(corpus), entry.Phrase, entry.Group, intent)

		resp, err := embedClient.Embed(t.Context(), entry.Phrase)
		if err != nil {
			t.Fatalf("embed phrase %q: %v", entry.Phrase, err)
		}

		byIntent[intent] = append(byIntent[intent], resp.Vector)
		t.Logf("  tokens=%d dim=%d", resp.InputTokens, len(resp.Vector))
	}

	// Compute mean vector per class.
	prototypes := make(map[llm.Intent][]float32, len(byIntent))
	for intent, vecs := range byIntent {
		mean := llm.MeanVector(vecs)
		prototypes[intent] = mean
		fmt.Printf("class=%s phrases=%d dim=%d\n", intent, len(vecs), len(mean))
	}

	if len(prototypes) == 0 {
		t.Fatal("no prototypes built — corpus may be empty or all groups unknown")
	}

	// Serialize to JSON.
	out, err := json.MarshalIndent(prototypes, "", "  ")
	if err != nil {
		t.Fatalf("marshal prototypes: %v", err)
	}

	outPath := prototypesPath()
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		t.Fatalf("create prototypes dir: %v", err)
	}
	if err := os.WriteFile(outPath, out, 0644); err != nil {
		t.Fatalf("write prototypes: %v", err)
	}

	t.Logf("prototypes written to %s (%d bytes)", outPath, len(out))
}

// prototypesPath returns the absolute path to the prototypes.json file.
func prototypesPath() string {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filename)
	return filepath.Join(base, "..", "..", "..", "finetune", "confidence-check", "data", "prototypes.json")
}
