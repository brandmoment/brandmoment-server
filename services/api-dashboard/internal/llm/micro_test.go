package llm

import (
	"context"
	"errors"
	"math"
	"testing"
)

// mockEmbedClient implements EmbedClient using a function field.
type mockEmbedClient struct {
	embedFn func(ctx context.Context, text string) (EmbedResponse, error)
}

func (m *mockEmbedClient) Embed(ctx context.Context, text string) (EmbedResponse, error) {
	return m.embedFn(ctx, text)
}

var _ EmbedClient = (*mockEmbedClient)(nil)

// ---- Cosine tests ----

func TestCosine(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
		eps  float64
	}{
		{
			name: "identical vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
			eps:  1e-6,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{0, 1, 0},
			want: 0.0,
			eps:  1e-6,
		},
		{
			name: "opposite vectors",
			a:    []float32{1, 0},
			b:    []float32{-1, 0},
			want: -1.0,
			eps:  1e-6,
		},
		{
			name: "zero vector a returns 0",
			a:    []float32{0, 0, 0},
			b:    []float32{1, 2, 3},
			want: 0.0,
			eps:  1e-6,
		},
		{
			name: "zero vector b returns 0",
			a:    []float32{1, 2, 3},
			b:    []float32{0, 0, 0},
			want: 0.0,
			eps:  1e-6,
		},
		{
			name: "both zero returns 0",
			a:    []float32{0, 0},
			b:    []float32{0, 0},
			want: 0.0,
			eps:  1e-6,
		},
		{
			name: "45 degree angle",
			a:    []float32{1, 1},
			b:    []float32{1, 0},
			want: math.Sqrt2 / 2,
			eps:  1e-6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Cosine(tt.a, tt.b)
			if math.Abs(got-tt.want) > tt.eps {
				t.Errorf("Cosine(%v, %v) = %f, want %f (eps=%g)", tt.a, tt.b, got, tt.want, tt.eps)
			}
		})
	}
}

func TestCosine_PanicOnDimMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on dimension mismatch, got none")
		}
	}()
	Cosine([]float32{1, 2}, []float32{1, 2, 3})
}

// ---- MeanVector tests ----

func TestMeanVector(t *testing.T) {
	t.Run("empty slice returns nil", func(t *testing.T) {
		got := MeanVector(nil)
		if got != nil {
			t.Errorf("MeanVector(nil) = %v, want nil", got)
		}
		got2 := MeanVector([][]float32{})
		if got2 != nil {
			t.Errorf("MeanVector([]) = %v, want nil", got2)
		}
	})

	t.Run("single vector returns copy", func(t *testing.T) {
		v := []float32{1.0, 2.0, 3.0}
		got := MeanVector([][]float32{v})
		if len(got) != len(v) {
			t.Fatalf("len = %d, want %d", len(got), len(v))
		}
		for i := range v {
			if math.Abs(float64(got[i]-v[i])) > 1e-6 {
				t.Errorf("[%d] = %f, want %f", i, got[i], v[i])
			}
		}
	})

	t.Run("mean of two vectors", func(t *testing.T) {
		a := []float32{0, 4}
		b := []float32{2, 0}
		got := MeanVector([][]float32{a, b})
		want := []float32{1, 2}
		for i, w := range want {
			if math.Abs(float64(got[i]-w)) > 1e-6 {
				t.Errorf("[%d] = %f, want %f", i, got[i], w)
			}
		}
	})

	t.Run("mean of three vectors", func(t *testing.T) {
		vecs := [][]float32{
			{3, 0, 6},
			{0, 3, 0},
			{0, 0, 3},
		}
		got := MeanVector(vecs)
		want := []float32{1, 1, 3}
		for i, w := range want {
			if math.Abs(float64(got[i]-w)) > 1e-6 {
				t.Errorf("[%d] = %f, want %f", i, got[i], w)
			}
		}
	})
}

func TestMeanVector_PanicOnDimMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on dimension mismatch, got none")
		}
	}()
	MeanVector([][]float32{{1, 2}, {1, 2, 3}})
}

// ---- EmbedMicro.Classify tests ----

// fixed vectors for intent prototypes:
//
//	valid:     (1, 0, 0)
//	ambiguous: (0, 1, 0)
//	gibberish: (0, 0, 1)
var testPrototypes = map[Intent][]float32{
	IntentValid:     {1, 0, 0},
	IntentAmbiguous: {0, 1, 0},
	IntentGibberish: {0, 0, 1},
}

func TestEmbedMicro_Classify(t *testing.T) {
	tests := []struct {
		name        string
		embedVec    []float32
		embedErr    error
		wantIntent  Intent
		wantErr     bool
		wantMargin  float64 // approximate
		wantScoreGT float64 // top1 must be > this
	}{
		{
			name:        "valid phrase — vector aligned with valid prototype",
			embedVec:    []float32{1, 0, 0},
			wantIntent:  IntentValid,
			wantMargin:  1.0, // cos(valid)=1, cos(ambiguous)=0, margin=1
			wantScoreGT: 0.9,
		},
		{
			name:        "gibberish phrase — vector aligned with gibberish prototype",
			embedVec:    []float32{0, 0, 1},
			wantIntent:  IntentGibberish,
			wantMargin:  1.0,
			wantScoreGT: 0.9,
		},
		{
			name:        "ambiguous phrase — vector aligned with ambiguous prototype",
			embedVec:    []float32{0, 1, 0},
			wantIntent:  IntentAmbiguous,
			wantMargin:  1.0,
			wantScoreGT: 0.9,
		},
		{
			name:        "near-valid phrase — close to valid but not perfect",
			embedVec:    []float32{0.9, 0.3, 0.1},
			wantIntent:  IntentValid,
			wantScoreGT: 0.8,
		},
		{
			name:     "embed client returns error",
			embedErr: errors.New("api timeout"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmbed := &mockEmbedClient{
				embedFn: func(ctx context.Context, text string) (EmbedResponse, error) {
					if tt.embedErr != nil {
						return EmbedResponse{}, tt.embedErr
					}
					return EmbedResponse{Vector: tt.embedVec, InputTokens: 10}, nil
				},
			}

			classifier := NewEmbedMicro(mockEmbed, testPrototypes)
			result, err := classifier.Classify(context.Background(), "test phrase")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Intent != tt.wantIntent {
				t.Errorf("Intent = %q, want %q", result.Intent, tt.wantIntent)
			}

			if result.Top1 < tt.wantScoreGT {
				t.Errorf("Top1 = %f, want > %f", result.Top1, tt.wantScoreGT)
			}

			if tt.wantMargin > 0 && math.Abs(result.Margin-tt.wantMargin) > 1e-5 {
				t.Errorf("Margin = %f, want ~%f", result.Margin, tt.wantMargin)
			}

			if result.InputTokens != 10 {
				t.Errorf("InputTokens = %d, want 10", result.InputTokens)
			}

			// Verify scores map contains all three intents.
			for _, intent := range []Intent{IntentValid, IntentAmbiguous, IntentGibberish} {
				if _, ok := result.Scores[intent]; !ok {
					t.Errorf("Scores missing intent %q", intent)
				}
			}
		})
	}
}

func TestEmbedMicro_Classify_ErrorPropagation(t *testing.T) {
	wantErr := errors.New("embedding service down")
	mockEmbed := &mockEmbedClient{
		embedFn: func(ctx context.Context, text string) (EmbedResponse, error) {
			return EmbedResponse{}, wantErr
		},
	}

	classifier := NewEmbedMicro(mockEmbed, testPrototypes)
	_, err := classifier.Classify(context.Background(), "any phrase")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error chain: got %v, want to contain %v", err, wantErr)
	}
}
