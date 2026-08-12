package lkrequest

import (
	"unsafe"
)

// MetricsSnapshot is a read-only snapshot of process-level runtime metrics.
//
// Each field is a cumulative count since the process started. Note: It only actually accumulates when the underlying lkrequest
// library is built with the telemetry feature; otherwise, all fields are always 0 (the call itself will still return successfully).
//
// The field layout of this struct strictly corresponds to the C-side lk_metrics_snapshot_t (6 uint64 values) for direct FFI filling.
type MetricsSnapshot struct {
	BytesIn           uint64
	BytesOut          uint64
	RequestsTotal     uint64
	RequestsFailed    uint64
	ActiveConnections uint64
	TotalConnections  uint64
}

// Metrics reads a snapshot of current process-level runtime metrics for integration with monitoring pipelines such as
// Prometheus / OpenTelemetry. This call is read-only and has minimal overhead.
func Metrics() (MetricsSnapshot, error) {
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_metrics_snapshot(uintptr(unsafe.Pointer(&cell[0])))
	if err := statusError(status, "metrics snapshot"); err != nil {
		return MetricsSnapshot{}, err
	}
	return *(*MetricsSnapshot)(unsafe.Pointer(&cell[0])), nil
}
