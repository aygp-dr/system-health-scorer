package main

import (
	"testing"
)

func TestScoreUtilization(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"zero", 0, 100},
		{"negative", -5, 100},
		{"30 pct", 30, 90},
		{"60 pct", 60, 80},
		{"70 pct", 70, 60},
		{"80 pct", 80, 40},
		{"90 pct", 90, 20},
		{"100 pct", 100, 0},
		{"over 100", 110, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreUtilization(tt.input)
			if absF(got-tt.expected) > 0.01 {
				t.Errorf("ScoreUtilization(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestScoreErrorCount(t *testing.T) {
	tests := []struct {
		name string
		input float64
		min  float64
		max  float64
	}{
		{"zero errors", 0, 100, 100},
		{"negative", -1, 100, 100},
		{"1 error", 1, 85, 95},
		{"10 errors", 10, 35, 38},
		{"50 errors", 50, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreErrorCount(tt.input)
			if got < tt.min || got > tt.max {
				t.Errorf("ScoreErrorCount(%v) = %v, want in [%v, %v]", tt.input, got, tt.min, tt.max)
			}
		})
	}
}

func TestScoreLatency(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"zero", 0, 100},
		{"negative", -10, 100},
		{"500ms", 500, 50},
		{"1000ms", 1000, 0},
		{"over 1000", 2000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreLatency(tt.input)
			if absF(got-tt.expected) > 0.01 {
				t.Errorf("ScoreLatency(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestComputeComponentScore(t *testing.T) {
	mw := MetricWeights{
		Utilization: 0.40,
		ErrorCount:  0.35,
		LatencyMs:   0.25,
	}

	t.Run("perfect metrics", func(t *testing.T) {
		r := MetricReading{Utilization: 0, ErrorCount: 0, LatencyMs: 0}
		score := ComputeComponentScore(r, mw)
		if absF(score-100) > 0.01 {
			t.Errorf("got %v, want 100", score)
		}
	})

	t.Run("worst metrics", func(t *testing.T) {
		r := MetricReading{Utilization: 100, ErrorCount: 100, LatencyMs: 1000}
		score := ComputeComponentScore(r, mw)
		if score > 1 {
			t.Errorf("got %v, want ~0", score)
		}
	})

	t.Run("mixed metrics", func(t *testing.T) {
		r := MetricReading{Utilization: 50, ErrorCount: 5, LatencyMs: 200}
		score := ComputeComponentScore(r, mw)
		if score < 30 || score > 90 {
			t.Errorf("got %v, want in [30, 90]", score)
		}
	})
}

func TestComputeOverallScore(t *testing.T) {
	t.Run("weighted average", func(t *testing.T) {
		components := []*Component{
			{Name: "a", Weight: 0.6, Score: 80},
			{Name: "b", Weight: 0.4, Score: 60},
		}
		got := ComputeOverallScore(components)
		expected := (80*0.6 + 60*0.4) / (0.6 + 0.4)
		if absF(got-expected) > 0.01 {
			t.Errorf("got %v, want %v", got, expected)
		}
	})

	t.Run("empty", func(t *testing.T) {
		got := ComputeOverallScore(nil)
		if got != 0 {
			t.Errorf("got %v, want 0", got)
		}
	})
}

func TestNewHealthScorer(t *testing.T) {
	hs := NewHealthScorer()

	if len(hs.Components) != 8 {
		t.Fatalf("expected 8 components, got %d", len(hs.Components))
	}

	var totalWeight float64
	for _, c := range hs.Components {
		if len(c.History) != hs.MaxHistory {
			t.Errorf("component %s: expected %d history, got %d", c.Name, hs.MaxHistory, len(c.History))
		}
		if c.Score < 0 || c.Score > 100 {
			t.Errorf("component %s: score %v out of [0, 100]", c.Name, c.Score)
		}
		totalWeight += c.Weight
	}

	if absF(totalWeight-1.0) > 0.01 {
		t.Errorf("weights sum to %v, want 1.0", totalWeight)
	}

	if hs.OverallScore < 0 || hs.OverallScore > 100 {
		t.Errorf("overall score %v out of [0, 100]", hs.OverallScore)
	}
}

func TestHealthScorerTick(t *testing.T) {
	hs := NewHealthScorer()
	hs.Tick()

	for _, c := range hs.Components {
		if c.Score < 0 || c.Score > 100 {
			t.Errorf("component %s: score %v out of range after tick", c.Name, c.Score)
		}
		if len(c.History) > hs.MaxHistory {
			t.Errorf("component %s: history %d exceeds max %d", c.Name, len(c.History), hs.MaxHistory)
		}
	}
}

func TestRenderBar(t *testing.T) {
	t.Run("full bar", func(t *testing.T) {
		bar := RenderBar(100, 100, 10)
		if len([]rune(bar)) != 10 {
			t.Errorf("expected 10 runes, got %d: %q", len([]rune(bar)), bar)
		}
	})

	t.Run("empty bar", func(t *testing.T) {
		bar := RenderBar(0, 100, 10)
		if len([]rune(bar)) != 10 {
			t.Errorf("expected 10 runes, got %d: %q", len([]rune(bar)), bar)
		}
	})

	t.Run("half bar", func(t *testing.T) {
		bar := RenderBar(50, 100, 10)
		if len([]rune(bar)) != 10 {
			t.Errorf("expected 10 runes, got %d: %q", len([]rune(bar)), bar)
		}
	})

	t.Run("zero max", func(t *testing.T) {
		if bar := RenderBar(50, 0, 10); bar != "" {
			t.Errorf("expected empty, got %q", bar)
		}
	})

	t.Run("zero width", func(t *testing.T) {
		if bar := RenderBar(50, 100, 0); bar != "" {
			t.Errorf("expected empty, got %q", bar)
		}
	})
}

func TestScoreLabel(t *testing.T) {
	tests := []struct {
		score float64
		label string
	}{
		{95, "Healthy"},
		{85, "Good"},
		{60, "Warning"},
		{40, "Degraded"},
		{20, "Critical"},
		{0, "Critical"},
	}
	for _, tt := range tests {
		got := ScoreLabel(tt.score)
		if got != tt.label {
			t.Errorf("ScoreLabel(%v) = %q, want %q", tt.score, got, tt.label)
		}
	}
}

func TestClampF(t *testing.T) {
	tests := []struct {
		v, lo, hi, want float64
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
		{10, 0, 10, 10},
	}
	for _, tt := range tests {
		got := clampF(tt.v, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clampF(%v, %v, %v) = %v, want %v", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
