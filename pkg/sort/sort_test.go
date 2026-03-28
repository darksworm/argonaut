package sort_test

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
	pkgsort "github.com/darksworm/argonaut/pkg/sort"
)

// testItem implements pkgsort.Sortable for use in tests.
type testItem struct {
	health string
	sync   string
	kind   string
	name   string
}

func (t testItem) SortKey() model.SortKey {
	return model.SortKey{Health: t.health, Sync: t.sync, Kind: t.kind, Name: t.name}
}

func names(items []testItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.name
	}
	return out
}

func TestSort_HealthAsc(t *testing.T) {
	items := []testItem{
		{health: "Healthy", name: "a"},
		{health: "Degraded", name: "b"},
		{health: "Progressing", name: "c"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	got := names(items)
	want := []string{"b", "c", "a"} // Degraded < Progressing < Healthy
	for i, n := range want {
		if got[i] != n {
			t.Errorf("HealthAsc pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

func TestSort_HealthDesc(t *testing.T) {
	items := []testItem{
		{health: "Healthy", name: "a"},
		{health: "Degraded", name: "b"},
		{health: "Progressing", name: "c"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortDesc})
	got := names(items)
	want := []string{"a", "c", "b"} // Healthy > Progressing > Degraded
	for i, n := range want {
		if got[i] != n {
			t.Errorf("HealthDesc pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

func TestSort_SyncAsc(t *testing.T) {
	items := []testItem{
		{sync: "Synced", name: "a"},
		{sync: "OutOfSync", name: "b"},
		{sync: "Unknown", name: "c"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldSync, Direction: model.SortAsc})
	got := names(items)
	want := []string{"b", "c", "a"} // OutOfSync < Unknown < Synced
	for i, n := range want {
		if got[i] != n {
			t.Errorf("SyncAsc pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

func TestSort_SyncDesc(t *testing.T) {
	items := []testItem{
		{sync: "Synced", name: "a"},
		{sync: "OutOfSync", name: "b"},
		{sync: "Unknown", name: "c"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldSync, Direction: model.SortDesc})
	got := names(items)
	want := []string{"a", "c", "b"} // Synced > Unknown > OutOfSync
	for i, n := range want {
		if got[i] != n {
			t.Errorf("SyncDesc pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

func TestSort_NameAsc(t *testing.T) {
	items := []testItem{
		{name: "zebra"},
		{name: "apple"},
		{name: "mango"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldName, Direction: model.SortAsc})
	got := names(items)
	want := []string{"apple", "mango", "zebra"}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("NameAsc pos %d: got %q want %q", i, got[i], n)
		}
	}
}

func TestSort_NameDesc(t *testing.T) {
	items := []testItem{
		{name: "zebra"},
		{name: "apple"},
		{name: "mango"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldName, Direction: model.SortDesc})
	got := names(items)
	want := []string{"zebra", "mango", "apple"}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("NameDesc pos %d: got %q want %q", i, got[i], n)
		}
	}
}

// TestSort_HealthTiebreakByName verifies that equal health values are broken by name.
func TestSort_HealthTiebreakByName(t *testing.T) {
	items := []testItem{
		{health: "Healthy", name: "zebra"},
		{health: "Degraded", name: "alpha"},
		{health: "Healthy", name: "apple"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	got := names(items)
	want := []string{"alpha", "apple", "zebra"} // Degraded first, then Healthy sorted by name asc
	for i, n := range want {
		if got[i] != n {
			t.Errorf("HealthTiebreakByName pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

// TestSort_SyncTiebreakByName verifies that equal sync values are broken by name.
func TestSort_SyncTiebreakByName(t *testing.T) {
	items := []testItem{
		{sync: "Synced", name: "zebra"},
		{sync: "OutOfSync", name: "alpha"},
		{sync: "Synced", name: "apple"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldSync, Direction: model.SortAsc})
	got := names(items)
	want := []string{"alpha", "apple", "zebra"} // OutOfSync first, then Synced sorted by name asc
	for i, n := range want {
		if got[i] != n {
			t.Errorf("SyncTiebreakByName pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

// TestSort_NameBeatsKindTiebreak verifies that name takes priority over kind when
// both the primary status field is equal and the two items have different names AND
// different kinds (in opposing alphabetical orders). Name must win, not kind.
func TestSort_NameBeatsKindTiebreak(t *testing.T) {
	// alpha has name "alpha" (early) but kind "Service" (late)
	// zebra has name "zebra" (late) but kind "Deployment" (early)
	// Equal health → name tiebreak should fire first: alpha < zebra → alpha wins.
	// If kind were compared first, Deployment < Service → zebra would win incorrectly.
	items := []testItem{
		{health: "Healthy", kind: "Service", name: "alpha"},
		{health: "Healthy", kind: "Deployment", name: "zebra"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	if items[0].name != "alpha" {
		t.Errorf("NameBeatsKindTiebreak: expected alpha first (name wins over kind), got %q", items[0].name)
	}
}

// TestSort_KindTiebreakAfterName verifies that when both status and name are equal,
// kind is used as the final tiebreak.
func TestSort_KindTiebreakAfterName(t *testing.T) {
	items := []testItem{
		{health: "Healthy", kind: "Service", name: "app"},
		{health: "Healthy", kind: "Deployment", name: "app"},
		{health: "Healthy", kind: "ConfigMap", name: "app"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	got := make([]string, len(items))
	for i, it := range items {
		got[i] = it.kind
	}
	want := []string{"ConfigMap", "Deployment", "Service"}
	for i, k := range want {
		if got[i] != k {
			t.Errorf("KindTiebreak pos %d: got %q want %q (full: %v)", i, got[i], k, got)
		}
	}
}

func TestSort_Empty(t *testing.T) {
	var items []testItem
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	// no panic, no-op
}

func TestSort_SingleItem(t *testing.T) {
	items := []testItem{{health: "Degraded", name: "only"}}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	if items[0].name != "only" {
		t.Errorf("single item changed: %v", items[0])
	}
}

// TestSort_CaseInsensitiveStatus verifies getStatusOrder tolerates casing variations.
func TestSort_CaseInsensitiveStatus(t *testing.T) {
	items := []testItem{
		{health: "healthy", name: "a"},
		{health: "DEGRADED", name: "b"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	if items[0].name != "b" {
		t.Errorf("expected DEGRADED first, got %q", items[0].name)
	}
}

// TestSort_HealthTiebreakByNameDesc verifies that the name tiebreak respects DESC
// direction: equal-health items should appear in reverse-name order.
func TestSort_HealthTiebreakByNameDesc(t *testing.T) {
	items := []testItem{
		{health: "Healthy", name: "apple"},
		{health: "Degraded", name: "mid"},
		{health: "Healthy", name: "zebra"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortDesc})
	got := names(items)
	want := []string{"zebra", "apple", "mid"} // Healthy first (desc), zebra > apple by name desc, then Degraded
	for i, n := range want {
		if got[i] != n {
			t.Errorf("HealthTiebreakByNameDesc pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

// TestSort_EmptyStatusTreatedAsUnknown verifies that an empty status string is ranked
// the same as Unknown (the defaultVal path in getStatusOrder).
func TestSort_EmptyStatusTreatedAsUnknown(t *testing.T) {
	items := []testItem{
		{health: "Healthy", name: "a"},
		{health: "", name: "b"},       // empty → ranks as Unknown (4)
		{health: "Unknown", name: "c"}, // explicit Unknown (4)
		{health: "Degraded", name: "d"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	got := names(items)
	// Degraded(0) < empty/Unknown(4) == Unknown(4) < Healthy(5)
	// empty and Unknown have the same rank; name tiebreak: b < c
	want := []string{"d", "b", "c", "a"}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("EmptyStatus pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}

// TestSort_AllHealthStatuses verifies the full rank table:
// Degraded < Missing < Progressing < Suspended < Unknown < Healthy.
func TestSort_AllHealthStatuses(t *testing.T) {
	items := []testItem{
		{health: "Healthy", name: "f"},
		{health: "Unknown", name: "e"},
		{health: "Suspended", name: "d"},
		{health: "Progressing", name: "c"},
		{health: "Missing", name: "b"},
		{health: "Degraded", name: "a"},
	}
	pkgsort.Sort(items, model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})
	got := names(items)
	want := []string{"a", "b", "c", "d", "e", "f"}
	for i, n := range want {
		if got[i] != n {
			t.Errorf("AllHealthStatuses pos %d: got %q want %q (full: %v)", i, got[i], n, got)
		}
	}
}
