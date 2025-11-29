package sort

import (
	"strings"

	"github.com/darksworm/argonaut/pkg/model"
)

// Semantic ordering for sync statuses (problems first when ascending)
var syncStatusOrder = map[string]int{
	"OutOfSync": 0,
	"Unknown":   1,
	"Synced":    2,
}

// Semantic ordering for health statuses (problems first when ascending)
var healthStatusOrder = map[string]int{
	"Degraded":    0,
	"Missing":     1,
	"Progressing": 2,
	"Suspended":   3,
	"Unknown":     4,
	"Healthy":     5,
}

// SortApps sorts apps according to the provided configuration using insertion sort.
// Uses semantic ordering for sync/health statuses and falls back to name for stability.
func SortApps(apps []model.App, config model.SortConfig) {
	if len(apps) <= 1 {
		return
	}

	less := comparator(config)

	// Insertion sort - efficient for small lists and maintains stability
	for i := 1; i < len(apps); i++ {
		j := i
		for j > 0 && less(apps[j], apps[j-1]) {
			apps[j-1], apps[j] = apps[j], apps[j-1]
			j--
		}
	}
}

// comparator returns a less function based on sort config
func comparator(config model.SortConfig) func(a, b model.App) bool {
	return func(a, b model.App) bool {
		cmp := compareByField(a, b, config.Field)

		// If primary field is equal, fall back to name for stability
		if cmp == 0 && config.Field != model.SortFieldName {
			cmp = strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		}

		// Apply direction
		if config.Direction == model.SortDesc {
			return cmp > 0
		}
		return cmp < 0
	}
}

// compareByField compares two apps by the specified field
// Returns negative if a < b, positive if a > b, zero if equal
func compareByField(a, b model.App, field model.SortField) int {
	switch field {
	case model.SortFieldSync:
		return compareSyncStatus(a.Sync, b.Sync)
	case model.SortFieldHealth:
		return compareHealthStatus(a.Health, b.Health)
	default: // name
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}
}

// compareSyncStatus compares sync statuses using semantic ordering
func compareSyncStatus(a, b string) int {
	orderA := getStatusOrder(syncStatusOrder, a, 1) // Unknown values get middle priority
	orderB := getStatusOrder(syncStatusOrder, b, 1)
	return orderA - orderB
}

// compareHealthStatus compares health statuses using semantic ordering
func compareHealthStatus(a, b string) int {
	orderA := getStatusOrder(healthStatusOrder, a, 4) // Unknown values treated as Unknown status
	orderB := getStatusOrder(healthStatusOrder, b, 4)
	return orderA - orderB
}

// getStatusOrder returns the order value for a status, using defaultVal for unknown statuses
func getStatusOrder(orderMap map[string]int, status string, defaultVal int) int {
	if order, ok := orderMap[status]; ok {
		return order
	}
	return defaultVal
}
