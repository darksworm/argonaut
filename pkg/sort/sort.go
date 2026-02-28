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

// Sortable is satisfied by types that expose a SortKey for semantic ordering.
type Sortable interface {
	SortKey() model.SortKey
}

// comparatorGeneric provides a less function for any type implementing Sortable.
// It applies semantic health/sync ordering and tiebreaks by kind then name.
func comparatorGeneric[T Sortable](config model.SortConfig) func(a, b T) bool {
	return func(a, b T) bool {
		ak := a.SortKey()
		bk := b.SortKey()
		var cmp int
		switch config.Field {
		case model.SortFieldHealth:
			cmp = compareHealthStatus(ak.Health, bk.Health)
		case model.SortFieldSync:
			cmp = compareSyncStatus(ak.Sync, bk.Sync)
		default:
			cmp = strings.Compare(strings.ToLower(ak.Name), strings.ToLower(bk.Name))
		}

		// If primary field is equal and not name, fall back to name for stability
		if cmp == 0 && config.Field != model.SortFieldName {
			cmp = strings.Compare(strings.ToLower(ak.Name), strings.ToLower(bk.Name))
		}

		if cmp != 0 {
			if config.Direction == model.SortDesc {
				return cmp > 0
			}
			return cmp < 0
		}

		// Tiebreak by (kind, name) case-insensitive
		cmp = strings.Compare(strings.ToLower(ak.Kind), strings.ToLower(bk.Kind))
		if cmp == 0 {
			cmp = strings.Compare(strings.ToLower(ak.Name), strings.ToLower(bk.Name))
		}

		if config.Direction == model.SortDesc {
			return cmp > 0
		}
		return cmp < 0
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
	s := strings.TrimSpace(status)
	if s == "" {
		return defaultVal
	}
	// Try exact match first
	if order, ok := orderMap[s]; ok {
		return order
	}
	// Fall back to case-insensitive match to tolerate API casing variations
	for k, order := range orderMap {
		if strings.EqualFold(k, s) {
			return order
		}
	}
	return defaultVal
}

// Sort sorts any slice whose element type implements Sortable using the same
// semantic health/sync ordering as SortApps. Tiebreaks always use (kind, name).
// Uses insertion sort; efficient for small sibling lists.
func Sort[T Sortable](items []T, config model.SortConfig) {
	if len(items) <= 1 {
		return
	}
	less := comparatorGeneric[T](config)

	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && less(items[j], items[j-1]) {
			items[j-1], items[j] = items[j], items[j-1]
			j--
		}
	}
}
