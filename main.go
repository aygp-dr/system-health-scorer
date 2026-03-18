package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	dashboardView viewState = iota
	detailView
	helpView
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	healthyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	goodStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	degradedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	criticalStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236"))
	helpTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	barStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
)

func scoreStyle(score float64) lipgloss.Style {
	switch {
	case score >= 90:
		return healthyStyle
	case score >= 70:
		return goodStyle
	case score >= 50:
		return warningStyle
	case score >= 30:
		return degradedStyle
	default:
		return criticalStyle
	}
}

type tickMsg time.Time

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type model struct {
	scorer   *HealthScorer
	view     viewState
	cursor   int
	width    int
	height   int
	tickRate time.Duration
}

func newModel() model {
	return model{
		scorer:   NewHealthScorer(),
		view:     dashboardView,
		cursor:   0,
		width:    80,
		height:   24,
		tickRate: 5 * time.Second,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd(m.tickRate)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		m.scorer.Tick()
		return m, tickCmd(m.tickRate)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			if m.view == helpView {
				m.view = dashboardView
			} else {
				m.view = helpView
			}
		case "esc", "backspace":
			if m.view != dashboardView {
				m.view = dashboardView
			}
		case "j", "down":
			if m.view == dashboardView && m.cursor < len(m.scorer.Components)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.view == dashboardView && m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.view == dashboardView {
				m.view = detailView
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.view {
	case detailView:
		return m.viewDetail()
	case helpView:
		return m.viewHelp()
	default:
		return m.viewDashboard()
	}
}

func (m model) viewDashboard() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("System Health Dashboard"))
	b.WriteString("\n\n")

	overall := m.scorer.OverallScore
	overallStr := fmt.Sprintf("Overall Health: %.1f/100 [%s]", overall, ScoreLabel(overall))
	b.WriteString(scoreStyle(overall).Render(overallStr))
	b.WriteString("\n\n")

	header := fmt.Sprintf("  %-12s %6s  %-20s  %s", "COMPONENT", "SCORE", "", "STATUS")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 58)))
	b.WriteString("\n")

	for i, c := range m.scorer.Components {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}
		bar := RenderBar(c.Score, 100, 20)
		style := scoreStyle(c.Score)

		if i == m.cursor {
			row := fmt.Sprintf("%s%-12s %5.1f  %s  %s",
				cursor, c.Name, c.Score, bar, ScoreLabel(c.Score))
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(fmt.Sprintf("%s%-12s %s  %s  %s",
				cursor,
				c.Name,
				style.Render(fmt.Sprintf("%5.1f", c.Score)),
				barStyle.Render(bar),
				style.Render(ScoreLabel(c.Score)),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpTextStyle.Render("j/k: navigate  enter: details  ?: help  q: quit"))
	return b.String()
}

func (m model) viewDetail() string {
	c := m.scorer.Components[m.cursor]
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("Component: %s", c.Name)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Health Score: %s  Weight: %.0f%%\n\n",
		scoreStyle(c.Score).Render(fmt.Sprintf("%.1f", c.Score)),
		c.Weight*100,
	))

	metrics := []struct {
		name string
		unit string
		max  float64
		get  func(MetricReading) float64
	}{
		{"Utilization", "%", 100, func(r MetricReading) float64 { return r.Utilization }},
		{"Error Count", "", 30, func(r MetricReading) float64 { return r.ErrorCount }},
		{"Latency", "ms", 500, func(r MetricReading) float64 { return r.LatencyMs }},
	}

	for _, metric := range metrics {
		b.WriteString(headerStyle.Render(fmt.Sprintf("  %s", metric.name)))
		b.WriteString("\n")
		for i, reading := range c.History {
			val := metric.get(reading)
			bar := RenderBar(val, metric.max, 30)
			label := fmt.Sprintf("  %2d │ ", i+1)
			valStr := fmt.Sprintf(" %6.1f%s", val, metric.unit)
			b.WriteString(dimStyle.Render(label))
			b.WriteString(barStyle.Render(bar))
			b.WriteString(dimStyle.Render(valStr))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(helpTextStyle.Render("esc: back  ?: help  q: quit"))
	return b.String()
}

func (m model) viewHelp() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	keys := []struct{ key, desc string }{
		{"j / ↓", "Move cursor down"},
		{"k / ↑", "Move cursor up"},
		{"enter", "View component details"},
		{"esc", "Back to dashboard"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("  %-10s  %s\n", headerStyle.Render(k.key), k.desc))
	}

	b.WriteString("\n")
	b.WriteString(helpTextStyle.Render("Press esc or ? to return"))
	return b.String()
}

// JSONOutput represents the health score data in JSON format.
type JSONOutput struct {
	Timestamp    string          `json:"timestamp"`
	OverallScore float64         `json:"overall_score"`
	OverallLabel string          `json:"overall_label"`
	Components   []JSONComponent `json:"components"`
}

// JSONComponent represents a single component in JSON output.
type JSONComponent struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Label       string  `json:"label"`
	Weight      float64 `json:"weight"`
	Utilization float64 `json:"utilization_pct"`
	ErrorCount  float64 `json:"error_count"`
	LatencyMs   float64 `json:"latency_ms"`
}

func main() {
	jsonFlag := flag.Bool("json", false, "Output health scores as JSON (non-interactive)")
	flag.Parse()

	if *jsonFlag {
		scorer := NewHealthScorer()
		out := JSONOutput{
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			OverallScore: math.Round(scorer.OverallScore*10) / 10,
			OverallLabel: ScoreLabel(scorer.OverallScore),
		}
		for _, c := range scorer.Components {
			latest := c.History[len(c.History)-1]
			out.Components = append(out.Components, JSONComponent{
				Name:        c.Name,
				Score:       math.Round(c.Score*10) / 10,
				Label:       ScoreLabel(c.Score),
				Weight:      c.Weight,
				Utilization: math.Round(latest.Utilization*10) / 10,
				ErrorCount:  math.Round(latest.ErrorCount*10) / 10,
				LatencyMs:   math.Round(latest.LatencyMs*10) / 10,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
