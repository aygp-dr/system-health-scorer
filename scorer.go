package main

import (
	"math"
	"math/rand"
	"time"
)

// MetricReading holds a single point-in-time measurement for a component.
type MetricReading struct {
	Timestamp   time.Time
	Utilization float64 // 0-100 percent
	ErrorCount  float64 // 0+ count per interval
	LatencyMs   float64 // 0+ milliseconds
}

// MetricWeights defines how much each metric contributes to a component's score.
type MetricWeights struct {
	Utilization float64
	ErrorCount  float64
	LatencyMs   float64
}

// Component represents a monitored system component.
type Component struct {
	Name    string
	Weight  float64 // contribution to overall score
	History []MetricReading
	Score   float64 // computed 0-100

	// simulation baselines
	baseUtil    float64
	baseErrors  float64
	baseLatency float64
}

// HealthScorer manages all components and computes health scores.
type HealthScorer struct {
	Components    []*Component
	OverallScore  float64
	MetricWeights MetricWeights
	MaxHistory    int
}

// NewHealthScorer creates a scorer with default components and weights.
func NewHealthScorer() *HealthScorer {
	hs := &HealthScorer{
		MetricWeights: MetricWeights{
			Utilization: 0.40,
			ErrorCount:  0.35,
			LatencyMs:   0.25,
		},
		MaxHistory: 10,
	}

	hs.Components = []*Component{
		{Name: "cpu", Weight: 0.15, baseUtil: 45, baseErrors: 0, baseLatency: 2},
		{Name: "memory", Weight: 0.12, baseUtil: 62, baseErrors: 0, baseLatency: 1},
		{Name: "disk", Weight: 0.10, baseUtil: 55, baseErrors: 1, baseLatency: 15},
		{Name: "network", Weight: 0.10, baseUtil: 30, baseErrors: 2, baseLatency: 25},
		{Name: "database", Weight: 0.20, baseUtil: 40, baseErrors: 1, baseLatency: 45},
		{Name: "cache", Weight: 0.08, baseUtil: 35, baseErrors: 0, baseLatency: 5},
		{Name: "queue", Weight: 0.07, baseUtil: 25, baseErrors: 3, baseLatency: 30},
		{Name: "api", Weight: 0.18, baseUtil: 50, baseErrors: 5, baseLatency: 120},
	}

	for i := 0; i < hs.MaxHistory; i++ {
		hs.Tick()
	}

	return hs
}

// ScoreUtilization scores utilization percentage (0-100) to a health score (0-100).
// Lower utilization = higher score, with steeper penalties above 60%.
func ScoreUtilization(util float64) float64 {
	if util <= 0 {
		return 100
	}
	if util >= 100 {
		return 0
	}
	if util <= 60 {
		return 100 - (util/60)*20
	}
	if util <= 80 {
		return 80 - ((util-60)/20)*40
	}
	return 40 - ((util-80)/20)*40
}

// ScoreErrorCount scores error count to a health score (0-100).
// Uses exponential decay: 0 errors = 100, ~23 errors ≈ 10.
func ScoreErrorCount(errors float64) float64 {
	if errors <= 0 {
		return 100
	}
	return 100 * math.Exp(-errors/10)
}

// ScoreLatency scores latency in ms to a health score (0-100).
// Linear from 100 (0ms) to 0 (>=1000ms).
func ScoreLatency(latencyMs float64) float64 {
	if latencyMs <= 0 {
		return 100
	}
	if latencyMs >= 1000 {
		return 0
	}
	return 100 * (1 - latencyMs/1000)
}

// ComputeComponentScore calculates a weighted health score from a metric reading.
func ComputeComponentScore(r MetricReading, mw MetricWeights) float64 {
	uScore := ScoreUtilization(r.Utilization)
	eScore := ScoreErrorCount(r.ErrorCount)
	lScore := ScoreLatency(r.LatencyMs)
	return uScore*mw.Utilization + eScore*mw.ErrorCount + lScore*mw.LatencyMs
}

// ComputeOverallScore calculates the weighted average of all component scores.
func ComputeOverallScore(components []*Component) float64 {
	var total, weightSum float64
	for _, c := range components {
		total += c.Score * c.Weight
		weightSum += c.Weight
	}
	if weightSum == 0 {
		return 0
	}
	return total / weightSum
}

// Tick generates a new metric reading for each component and recomputes scores.
func (hs *HealthScorer) Tick() {
	now := time.Now()
	for _, c := range hs.Components {
		reading := MetricReading{
			Timestamp:   now,
			Utilization: clampF(c.baseUtil+rand.Float64()*20-10, 0, 100),
			ErrorCount:  math.Max(0, c.baseErrors+rand.Float64()*6-3),
			LatencyMs:   math.Max(0, c.baseLatency+rand.Float64()*40-20),
		}
		c.History = append(c.History, reading)
		if len(c.History) > hs.MaxHistory {
			c.History = c.History[len(c.History)-hs.MaxHistory:]
		}
		c.Score = ComputeComponentScore(reading, hs.MetricWeights)
	}
	hs.OverallScore = ComputeOverallScore(hs.Components)
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// RenderBar renders a horizontal bar chart using Unicode block characters.
func RenderBar(value, maxValue float64, width int) string {
	if maxValue <= 0 || width <= 0 {
		return ""
	}
	blocks := []rune{'░', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}
	ratio := value / maxValue
	if ratio > 1 {
		ratio = 1
	}
	if ratio < 0 {
		ratio = 0
	}
	fullBlocks := int(ratio * float64(width))
	remainder := ratio*float64(width) - float64(fullBlocks)

	bar := make([]rune, 0, width)
	for i := 0; i < fullBlocks && i < width; i++ {
		bar = append(bar, blocks[8])
	}
	if fullBlocks < width {
		idx := int(remainder * 8)
		if idx > 0 {
			bar = append(bar, blocks[idx])
		}
	}
	for len(bar) < width {
		bar = append(bar, ' ')
	}
	return string(bar[:width])
}

// ScoreLabel returns a text label for a health score.
func ScoreLabel(score float64) string {
	switch {
	case score >= 90:
		return "Healthy"
	case score >= 70:
		return "Good"
	case score >= 50:
		return "Warning"
	case score >= 30:
		return "Degraded"
	default:
		return "Critical"
	}
}
