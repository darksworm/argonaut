//go:build e2e && unix

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestRealArgoMarkWaitsForNextSecond(t *testing.T) {
	for _, offset := range []time.Duration{0, 100 * time.Millisecond, 900 * time.Millisecond} {
		t.Run(offset.String(), func(t *testing.T) {
			prior := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
			clock := prior.Add(offset)
			since := markWithClock(func() time.Time { return clock }, func(d time.Duration) { clock = clock.Add(d) })
			if !since.After(prior) {
				t.Fatalf("mark %s still accepts an operation from the prior second", since)
			}
			if clock.Before(since) {
				t.Fatalf("returned before the clock reached mark %s", since)
			}
			if !since.Equal(prior.Add(time.Second)) {
				t.Fatalf("expected next second, got %s", since)
			}
		})
	}
}

func TestRealArgoOperationIgnoresPreviousSecond(t *testing.T) {
	prior := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	clock := prior.Add(900 * time.Millisecond)
	since := markWithClock(func() time.Time { return clock }, func(d time.Duration) { clock = clock.Add(d) })
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := prior
		if requests.Add(1) > 1 {
			started = prior.Add(time.Second)
		}
		fmt.Fprintf(w, `{"status":{"operationState":{"startedAt":%q,"phase":"Succeeded"}}}`, started.Format(time.RFC3339))
	}))
	defer srv.Close()
	client, err := newRealArgo(&model.Server{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	app := client.waitForOperationAfter(t, "demo", since)
	if got := digString(app, "status", "operationState", "startedAt"); got != prior.Add(time.Second).Format(time.RFC3339) {
		t.Fatalf("accepted stale operation from %s", got)
	}
}
