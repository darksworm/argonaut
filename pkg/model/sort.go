package model

// SortField represents the field to sort applications by
type SortField string

const (
	SortFieldName   SortField = "name"
	SortFieldSync   SortField = "sync"
	SortFieldHealth SortField = "health"
)

// SortDirection represents the sort direction
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// SortConfig holds the complete sort configuration
type SortConfig struct {
	Field     SortField     `json:"field"`
	Direction SortDirection `json:"direction"`
}

// DefaultSortConfig returns the default sort configuration
func DefaultSortConfig() SortConfig {
	return SortConfig{
		Field:     SortFieldName,
		Direction: SortAsc,
	}
}

// ValidSortFields returns all valid sort field values
func ValidSortFields() []SortField {
	return []SortField{SortFieldName, SortFieldSync, SortFieldHealth}
}

// ValidSortDirections returns all valid sort direction values
func ValidSortDirections() []SortDirection {
	return []SortDirection{SortAsc, SortDesc}
}

// IsValidSortField checks if a string is a valid sort field
func IsValidSortField(s string) bool {
	for _, f := range ValidSortFields() {
		if string(f) == s {
			return true
		}
	}
	return false
}

// IsValidSortDirection checks if a string is a valid sort direction
func IsValidSortDirection(s string) bool {
	for _, d := range ValidSortDirections() {
		if string(d) == s {
			return true
		}
	}
	return false
}

// Indicator returns the arrow character for the sort direction
func (d SortDirection) Indicator() string {
	if d == SortDesc {
		return "▼"
	}
	return "▲"
}

// SortKey holds the values used for semantic ordering.
type SortKey struct {
	Health string
	Sync   string
	Kind   string
	Name   string
}
