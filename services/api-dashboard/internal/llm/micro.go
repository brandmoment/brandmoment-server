package llm

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Intent represents a Level-1 intent class.
type Intent string

const (
	// IntentValid means the phrase is parseable into a PublisherRule.
	IntentValid Intent = "valid"
	// IntentAmbiguous means the phrase is partially expressible or has unclear scope.
	IntentAmbiguous Intent = "ambiguous"
	// IntentGibberish means the phrase is nonsense, out-of-scope, or contradictory.
	IntentGibberish Intent = "gibberish"
)

// MicroResult is the outcome of a MicroClassifier.Classify call.
type MicroResult struct {
	// Intent is the top-1 predicted intent class.
	Intent Intent
	// Top1 is the cosine similarity to the top-1 prototype.
	Top1 float64
	// Margin is Top1 minus the second-best cosine similarity.
	Margin float64
	// Scores holds cosine similarity for every intent prototype.
	Scores map[Intent]float64
	// LatencyMS is the wall-clock time for the classify call in milliseconds.
	LatencyMS float64
	// InputTokens is the number of tokens consumed by the embedding call.
	InputTokens int
}

// MicroClassifier is the interface for Level-1 intent classification.
type MicroClassifier interface {
	// Classify returns an intent prediction for the given phrase.
	Classify(ctx context.Context, phrase string) (MicroResult, error)
}

// EmbedMicro implements MicroClassifier using embedding cosine similarity
// against a fixed set of prototype vectors (one per intent class).
type EmbedMicro struct {
	client     EmbedClient
	prototypes map[Intent][]float32
}

// NewEmbedMicro creates an EmbedMicro. prototypes must contain at least one intent.
func NewEmbedMicro(client EmbedClient, prototypes map[Intent][]float32) *EmbedMicro {
	return &EmbedMicro{
		client:     client,
		prototypes: prototypes,
	}
}

// Classify embeds the phrase and computes cosine similarity against each prototype.
// It returns the top-1 intent, its score, the margin vs top-2, and all scores.
func (m *EmbedMicro) Classify(ctx context.Context, phrase string) (MicroResult, error) {
	start := time.Now()

	resp, err := m.client.Embed(ctx, phrase)
	if err != nil {
		return MicroResult{}, fmt.Errorf("micro classify embed: %w", err)
	}

	scores := make(map[Intent]float64, len(m.prototypes))
	for intent, proto := range m.prototypes {
		scores[intent] = Cosine(resp.Vector, proto)
	}

	var top1Intent, top2Intent Intent
	var top1Score, top2Score float64 = -2, -2

	for intent, score := range scores {
		if score > top1Score {
			top2Score = top1Score
			top2Intent = top1Intent
			top1Score = score
			top1Intent = intent
		} else if score > top2Score {
			top2Score = score
			top2Intent = intent
		}
	}

	margin := 0.0
	if top2Intent != "" {
		margin = top1Score - top2Score
	}

	latencyMS := float64(time.Since(start).Milliseconds())

	slog.InfoContext(ctx, "micro classify done",
		slog.String("intent", string(top1Intent)),
		slog.Float64("top1", top1Score),
		slog.Float64("margin", margin),
		slog.Float64("latency_ms", latencyMS),
		slog.Int("input_tokens", resp.InputTokens),
	)

	return MicroResult{
		Intent:      top1Intent,
		Top1:        top1Score,
		Margin:      margin,
		Scores:      scores,
		LatencyMS:   latencyMS,
		InputTokens: resp.InputTokens,
	}, nil
}

// MeanVector computes the element-wise mean of a slice of vectors.
// All vectors must have the same dimension; if the slice is empty, returns nil.
// Panics if vectors have differing dimensions (programming error — caller must ensure uniformity).
func MeanVector(vecs [][]float32) []float32 {
	if len(vecs) == 0 {
		return nil
	}
	dim := len(vecs[0])
	mean := make([]float32, dim)
	for _, v := range vecs {
		if len(v) != dim {
			panic(fmt.Sprintf("MeanVector: dimension mismatch: expected %d, got %d", dim, len(v)))
		}
		for i, x := range v {
			mean[i] += x
		}
	}
	n := float32(len(vecs))
	for i := range mean {
		mean[i] /= n
	}
	return mean
}

// Cosine returns the cosine similarity between two float32 vectors.
// Returns 0 if either vector is all-zeros (undefined direction).
// Panics if vectors have differing lengths.
func Cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		panic(fmt.Sprintf("Cosine: dimension mismatch: len(a)=%d len(b)=%d", len(a), len(b)))
	}
	var dot, normA, normB float64
	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
