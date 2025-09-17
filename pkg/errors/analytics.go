package errors

import (
	"context"
	"encoding/json"
	cblog "github.com/charmbracelet/log"
	"sort"
	"sync"
	"time"
)

// ErrorAnalytics provides error monitoring and analysis capabilities
type ErrorAnalytics struct {
	mu           sync.RWMutex
	errorHistory []ErrorRecord
	patterns     map[string]*ErrorPattern
	metrics      ErrorMetrics
	maxHistory   int
}

// ErrorRecord represents a single error occurrence
type ErrorRecord struct {
	Timestamp   time.Time              `json:"timestamp"`
	Category    ErrorCategory          `json:"category"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Severity    ErrorSeverity          `json:"severity"`
	Recoverable bool                   `json:"recoverable"`
	Context     map[string]interface{} `json:"context,omitempty"`
	UserAction  string                 `json:"userAction,omitempty"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolvedAt,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
}

// ErrorPattern tracks recurring error patterns
type ErrorPattern struct {
	Category     ErrorCategory `json:"category"`
	Code         string        `json:"code"`
	Count        int           `json:"count"`
	FirstSeen    time.Time     `json:"firstSeen"`
	LastSeen     time.Time     `json:"lastSeen"`
	Frequency    float64       `json:"frequency"` // errors per hour
	AvgDuration  time.Duration `json:"avgDuration,omitempty"`
	Trend        string        `json:"trend"` // "increasing", "stable", "decreasing"
	Severity     ErrorSeverity `json:"severity"`
	RecoveryRate float64       `json:"recoveryRate"` // percentage of errors that resolved
}

// ErrorMetrics provides aggregate error statistics
type ErrorMetrics struct {
	TotalErrors      int                   `json:"totalErrors"`
	ErrorsByCategory map[ErrorCategory]int `json:"errorsByCategory"`
	ErrorsBySeverity map[ErrorSeverity]int `json:"errorsBySeverity"`
	RecoveryRate     float64               `json:"recoveryRate"`
	AvgResolution    time.Duration         `json:"avgResolution"`
	TopPatterns      []*ErrorPattern       `json:"topPatterns"`
	TrendAnalysis    *ErrorTrend           `json:"trendAnalysis"`
	LastUpdated      time.Time             `json:"lastUpdated"`
	TimeWindow       time.Duration         `json:"timeWindow"`
}

// ErrorTrend analyzes error trends over time
type ErrorTrend struct {
	Direction          string   `json:"direction"`       // "improving", "stable", "degrading"
	ChangeRate         float64  `json:"changeRate"`      // percentage change
	PredictedErrors    int      `json:"predictedErrors"` // predicted errors in next hour
	Confidence         float64  `json:"confidence"`      // prediction confidence (0-1)
	RecommendedActions []string `json:"recommendedActions"`
}

// PredictiveAlert represents a predictive error alert
type PredictiveAlert struct {
	Pattern           string        `json:"pattern"`
	Probability       float64       `json:"probability"`
	ExpectedTime      time.Time     `json:"expectedTime"`
	Severity          ErrorSeverity `json:"severity"`
	PreventionActions []string      `json:"preventionActions"`
}

// NewErrorAnalytics creates a new error analytics system
func NewErrorAnalytics(maxHistory int) *ErrorAnalytics {
	if maxHistory <= 0 {
		maxHistory = 1000 // Default to 1000 records
	}

	return &ErrorAnalytics{
		errorHistory: make([]ErrorRecord, 0, maxHistory),
		patterns:     make(map[string]*ErrorPattern),
		metrics: ErrorMetrics{
			ErrorsByCategory: make(map[ErrorCategory]int),
			ErrorsBySeverity: make(map[ErrorSeverity]int),
			TimeWindow:       24 * time.Hour,
		},
		// Logger removed to avoid import cycle
		maxHistory: maxHistory,
	}
}

// RecordError records an error for analysis
func (ea *ErrorAnalytics) RecordError(err *ArgonautError) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	record := ErrorRecord{
		Timestamp:   time.Now(),
		Category:    err.Category,
		Code:        err.Code,
		Message:     err.Message,
		Severity:    err.Severity,
		Recoverable: err.Recoverable,
		Context:     err.Context,
		UserAction:  err.UserAction,
		Resolved:    false,
	}

	// Add to history
	ea.errorHistory = append(ea.errorHistory, record)

	// Limit history size
	if len(ea.errorHistory) > ea.maxHistory {
		ea.errorHistory = ea.errorHistory[1:]
	}

	// Update patterns
	ea.updatePattern(&record)

	// Update metrics
	ea.updateMetrics()

	cblog.With("component", "errors").Debug("Recorded error", "category", err.Category, "code", err.Code)
}

// RecordResolution records when an error is resolved
func (ea *ErrorAnalytics) RecordResolution(category ErrorCategory, code string, duration time.Duration) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	// Find the most recent unresolved error matching this pattern
	for i := len(ea.errorHistory) - 1; i >= 0; i-- {
		record := &ea.errorHistory[i]
		if record.Category == category && record.Code == code && !record.Resolved {
			resolvedAt := time.Now()
			record.Resolved = true
			record.ResolvedAt = &resolvedAt
			record.Duration = &duration

			cblog.With("component", "errors").Debug("Recorded resolution",
				"category", category, "code", code, "duration", duration)
			break
		}
	}

	// Update metrics
	ea.updateMetrics()
}

// updatePattern updates error patterns with new data
func (ea *ErrorAnalytics) updatePattern(record *ErrorRecord) {
	patternKey := string(record.Category) + "/" + record.Code

	pattern, exists := ea.patterns[patternKey]
	if !exists {
		pattern = &ErrorPattern{
			Category:  record.Category,
			Code:      record.Code,
			FirstSeen: record.Timestamp,
			Severity:  record.Severity,
		}
		ea.patterns[patternKey] = pattern
	}

	pattern.Count++
	pattern.LastSeen = record.Timestamp

	// Calculate frequency (errors per hour)
	duration := pattern.LastSeen.Sub(pattern.FirstSeen)
	if duration > 0 {
		pattern.Frequency = float64(pattern.Count) / duration.Hours()
	}

	// Update recovery rate
	resolved := 0
	total := 0
	var totalDuration time.Duration
	resolvedCount := 0

	for _, h := range ea.errorHistory {
		if h.Category == record.Category && h.Code == record.Code {
			total++
			if h.Resolved {
				resolved++
				if h.Duration != nil {
					totalDuration += *h.Duration
					resolvedCount++
				}
			}
		}
	}

	if total > 0 {
		pattern.RecoveryRate = float64(resolved) / float64(total)
	}

	if resolvedCount > 0 {
		pattern.AvgDuration = totalDuration / time.Duration(resolvedCount)
	}

	// Calculate trend
	pattern.Trend = ea.calculateTrend(patternKey)
}

// calculateTrend calculates the trend for an error pattern
func (ea *ErrorAnalytics) calculateTrend(patternKey string) string {
	// Simple trend calculation based on recent vs older occurrences
	now := time.Now()
	recent := 0
	older := 0

	for _, record := range ea.errorHistory {
		if string(record.Category)+"/"+record.Code == patternKey {
			age := now.Sub(record.Timestamp)
			if age <= time.Hour {
				recent++
			} else if age <= 6*time.Hour {
				older++
			}
		}
	}

	if recent > older*2 {
		return "increasing"
	} else if older > recent*2 {
		return "decreasing"
	}
	return "stable"
}

// updateMetrics updates aggregate error metrics
func (ea *ErrorAnalytics) updateMetrics() {
	now := time.Now()
	windowStart := now.Add(-ea.metrics.TimeWindow)

	// Reset counters
	ea.metrics.TotalErrors = 0
	ea.metrics.ErrorsByCategory = make(map[ErrorCategory]int)
	ea.metrics.ErrorsBySeverity = make(map[ErrorSeverity]int)

	totalResolved := 0
	var totalResolutionTime time.Duration
	resolvedWithTime := 0

	// Count errors within time window
	for _, record := range ea.errorHistory {
		if record.Timestamp.After(windowStart) {
			ea.metrics.TotalErrors++
			ea.metrics.ErrorsByCategory[record.Category]++
			ea.metrics.ErrorsBySeverity[record.Severity]++

			if record.Resolved {
				totalResolved++
				if record.Duration != nil {
					totalResolutionTime += *record.Duration
					resolvedWithTime++
				}
			}
		}
	}

	// Calculate recovery rate
	if ea.metrics.TotalErrors > 0 {
		ea.metrics.RecoveryRate = float64(totalResolved) / float64(ea.metrics.TotalErrors)
	}

	// Calculate average resolution time
	if resolvedWithTime > 0 {
		ea.metrics.AvgResolution = totalResolutionTime / time.Duration(resolvedWithTime)
	}

	// Update top patterns
	ea.updateTopPatterns()

	// Update trend analysis
	ea.updateTrendAnalysis()

	ea.metrics.LastUpdated = now
}

// updateTopPatterns updates the list of top error patterns
func (ea *ErrorAnalytics) updateTopPatterns() {
	patterns := make([]*ErrorPattern, 0, len(ea.patterns))
	for _, pattern := range ea.patterns {
		patterns = append(patterns, pattern)
	}

	// Sort by frequency (descending)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	// Take top 10
	if len(patterns) > 10 {
		patterns = patterns[:10]
	}

	ea.metrics.TopPatterns = patterns
}

// updateTrendAnalysis updates the trend analysis
func (ea *ErrorAnalytics) updateTrendAnalysis() {
	now := time.Now()
	hour1 := now.Add(-time.Hour)
	hour2 := now.Add(-2 * time.Hour)

	recentCount := 0
	previousCount := 0

	for _, record := range ea.errorHistory {
		if record.Timestamp.After(hour1) {
			recentCount++
		} else if record.Timestamp.After(hour2) {
			previousCount++
		}
	}

	trend := &ErrorTrend{
		Direction:          "stable",
		ChangeRate:         0,
		PredictedErrors:    recentCount,
		Confidence:         0.5,
		RecommendedActions: []string{},
	}

	if previousCount > 0 {
		trend.ChangeRate = float64(recentCount-previousCount) / float64(previousCount) * 100

		if trend.ChangeRate > 20 {
			trend.Direction = "degrading"
			trend.Confidence = 0.7
			trend.RecommendedActions = append(trend.RecommendedActions,
				"Investigate recent changes", "Review error patterns", "Consider increasing monitoring")
		} else if trend.ChangeRate < -20 {
			trend.Direction = "improving"
			trend.Confidence = 0.7
			trend.RecommendedActions = append(trend.RecommendedActions,
				"Continue current practices", "Document improvements")
		}
	}

	ea.metrics.TrendAnalysis = trend
}

// GetMetrics returns current error metrics
func (ea *ErrorAnalytics) GetMetrics() ErrorMetrics {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	return ea.metrics
}

// GetTopPatterns returns the most frequent error patterns
func (ea *ErrorAnalytics) GetTopPatterns(limit int) []*ErrorPattern {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	patterns := ea.metrics.TopPatterns
	if limit > 0 && len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns
}

// GeneratePredictiveAlerts generates alerts for likely future errors
func (ea *ErrorAnalytics) GeneratePredictiveAlerts() []PredictiveAlert {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	alerts := []PredictiveAlert{}

	for _, pattern := range ea.patterns {
		if pattern.Trend == "increasing" && pattern.Frequency > 1.0 { // More than 1 error per hour
			probability := pattern.Frequency / 10.0 // Simple probability calculation
			if probability > 1.0 {
				probability = 1.0
			}

			if probability > 0.3 { // 30% threshold
				alert := PredictiveAlert{
					Pattern:           string(pattern.Category) + "/" + pattern.Code,
					Probability:       probability,
					ExpectedTime:      time.Now().Add(time.Duration(60/pattern.Frequency) * time.Minute),
					Severity:          pattern.Severity,
					PreventionActions: ea.generatePreventionActions(pattern),
				}
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts
}

// generatePreventionActions generates prevention actions for an error pattern
func (ea *ErrorAnalytics) generatePreventionActions(pattern *ErrorPattern) []string {
	actions := []string{}

	switch pattern.Category {
	case ErrorNetwork:
		actions = append(actions, "Check network connectivity", "Verify server endpoints")
	case ErrorAuth:
		actions = append(actions, "Refresh authentication tokens", "Verify credentials")
	case ErrorAPI:
		actions = append(actions, "Check API server health", "Verify API endpoints")
	case ErrorTimeout:
		actions = append(actions, "Increase timeout values", "Optimize queries")
	default:
		actions = append(actions, "Review recent changes", "Check system resources")
	}

	if pattern.RecoveryRate < 0.5 {
		actions = append(actions, "Improve error handling", "Add more recovery mechanisms")
	}

	return actions
}

// GenerateReport generates a comprehensive error analysis report
func (ea *ErrorAnalytics) GenerateReport() map[string]interface{} {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"totalErrors":   ea.metrics.TotalErrors,
			"recoveryRate":  ea.metrics.RecoveryRate,
			"avgResolution": ea.metrics.AvgResolution.String(),
			"timeWindow":    ea.metrics.TimeWindow.String(),
			"lastUpdated":   ea.metrics.LastUpdated,
		},
		"categoryBreakdown": ea.metrics.ErrorsByCategory,
		"severityBreakdown": ea.metrics.ErrorsBySeverity,
		"topPatterns":       ea.metrics.TopPatterns,
		"trendAnalysis":     ea.metrics.TrendAnalysis,
		"predictiveAlerts":  ea.GeneratePredictiveAlerts(),
		"recommendations":   ea.generateRecommendations(),
	}
}

// generateRecommendations generates system-wide recommendations
func (ea *ErrorAnalytics) generateRecommendations() []string {
	recommendations := []string{}

	if ea.metrics.RecoveryRate < 0.5 {
		recommendations = append(recommendations,
			"Low recovery rate detected - improve error handling and retry mechanisms")
	}

	if ea.metrics.TrendAnalysis != nil && ea.metrics.TrendAnalysis.Direction == "degrading" {
		recommendations = append(recommendations,
			"Error rate is increasing - investigate recent changes and system health")
	}

	networkErrors := ea.metrics.ErrorsByCategory[ErrorNetwork]
	if networkErrors > ea.metrics.TotalErrors/3 {
		recommendations = append(recommendations,
			"High network error rate - check connectivity and server health")
	}

	authErrors := ea.metrics.ErrorsByCategory[ErrorAuth]
	if authErrors > 0 {
		recommendations = append(recommendations,
			"Authentication errors detected - verify credentials and token refresh")
	}

	return recommendations
}

// ExportMetrics exports metrics in JSON format
func (ea *ErrorAnalytics) ExportMetrics() ([]byte, error) {
	report := ea.GenerateReport()
	return json.MarshalIndent(report, "", "  ")
}

// Cleanup removes old error records beyond the time window
func (ea *ErrorAnalytics) Cleanup(ctx context.Context) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	cutoff := time.Now().Add(-ea.metrics.TimeWindow * 2) // Keep 2x the time window
	newHistory := make([]ErrorRecord, 0, len(ea.errorHistory))

	for _, record := range ea.errorHistory {
		if record.Timestamp.After(cutoff) {
			newHistory = append(newHistory, record)
		}
	}

	removed := len(ea.errorHistory) - len(newHistory)
	ea.errorHistory = newHistory

	if removed > 0 {
		cblog.With("component", "errors").Debug("Cleaned up old error records", "count", removed)
	}
}
