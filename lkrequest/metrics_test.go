package lkrequest

import "testing"

// TestMetrics verifies that Metrics() can successfully read the process-level metric snapshot. When the telemetry feature is not enabled,
// each field is 0. Here, only the successful call and the readability of the fields are verified.
func TestMetrics(t *testing.T) {
	snap, err := Metrics()
	if err != nil {
		t.Fatalf("Metrics() error = %v", err)
	}

	total := snap.BytesIn + snap.BytesOut + snap.RequestsTotal +
		snap.RequestsFailed + snap.ActiveConnections + snap.TotalConnections
	if total != 0 && snap.RequestsTotal < snap.RequestsFailed {
		t.Errorf("inconsistent snapshot: failed(%d) > total(%d)",
			snap.RequestsFailed, snap.RequestsTotal)
	}
}
