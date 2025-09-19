package errors

import (
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
