package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

// wireEvent is the subset of a Kubernetes Event we render. Core/v1 events
// carry count/lastTimestamp; events.k8s.io-style events carry
// eventTime/series instead, hence the normalization fallbacks.
type wireEvent struct {
	Type          string    `json:"type"`
	Reason        string    `json:"reason"`
	Message       string    `json:"message"`
	Count         int       `json:"count"`
	LastTimestamp time.Time `json:"lastTimestamp"`
	EventTime     time.Time `json:"eventTime"`
	Series        *struct {
		Count int `json:"count"`
	} `json:"series"`
}

// flattenWhitespace collapses all whitespace runs (newlines, tabs, spaces)
// into single spaces so a message renders as one wrappable line — embedded
// control characters would break the pane frame's row accounting.
func flattenWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (e wireEvent) normalize() model.ResourceEvent {
	lastSeen := e.LastTimestamp
	if lastSeen.IsZero() {
		lastSeen = e.EventTime
	}
	count := e.Count
	if count == 0 && e.Series != nil {
		count = e.Series.Count
	}
	if count == 0 {
		count = 1
	}
	return model.ResourceEvent{
		Type:     e.Type,
		Reason:   e.Reason,
		Message:  flattenWhitespace(e.Message),
		Count:    count,
		LastSeen: lastSeen,
	}
}

// ListEventsParams identifies whose events to fetch: the application itself,
// or one of its resources when all three Resource fields are set.
type ListEventsParams struct {
	AppName           string
	AppNamespace      string // "" = unset
	ResourceUID       string
	ResourceName      string
	ResourceNamespace string
}

// ListEvents fetches Kubernetes events for an application or one of its
// resources, normalized and sorted newest-first.
func (s *ApplicationService) ListEvents(ctx context.Context, params ListEventsParams) ([]model.ResourceEvent, error) {
	// The ArgoCD API resolves a resource by uid+name; the namespace may be
	// empty for cluster-scoped resources (verified against a live server).
	// A uid or name alone is rejected upstream, so catch it before HTTP.
	scoped := params.ResourceUID != "" && params.ResourceName != ""
	unscoped := params.ResourceUID == "" && params.ResourceName == "" && params.ResourceNamespace == ""
	if !scoped && !unscoped {
		return nil, fmt.Errorf("resource events require uid and name (got uid=%q name=%q namespace=%q)",
			params.ResourceUID, params.ResourceName, params.ResourceNamespace)
	}

	endpoint := fmt.Sprintf("/api/v1/applications/%s/events", url.PathEscape(params.AppName))

	query := url.Values{}
	if params.AppNamespace != "" {
		query.Set("appNamespace", params.AppNamespace)
	}
	if params.ResourceUID != "" {
		query.Set("resourceUID", params.ResourceUID)
		query.Set("resourceName", params.ResourceName)
		query.Set("resourceNamespace", params.ResourceNamespace)
	}
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	data, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to list events for application %s: %w", params.AppName, err)
	}

	// Accept both {items:[...]} and a bare array
	var eventList struct {
		Items []wireEvent `json:"items"`
	}
	if err := json.Unmarshal(data, &eventList); err != nil {
		if err := json.Unmarshal(data, &eventList.Items); err != nil {
			return nil, fmt.Errorf("failed to parse events response: %w", err)
		}
	}

	events := make([]model.ResourceEvent, 0, len(eventList.Items))
	for _, item := range eventList.Items {
		events = append(events, item.normalize())
	}
	SortEventsNewestFirst(events)
	return events, nil
}

// SortEventsNewestFirst sorts events in place, most recently seen first.
// Stable so that equally-timestamped events keep their server order.
func SortEventsNewestFirst(events []model.ResourceEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].LastSeen.After(events[j].LastSeen)
	})
}
