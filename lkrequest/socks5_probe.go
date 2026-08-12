package lkrequest

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// Socks5UDPProbeConfig configures a SOCKS5 UDP relay probe.
//
// When DNSServerHost is non-empty the probe encodes that host as the SOCKS5
// UDP target name (ATYP=domain). Otherwise DNSServerAddr must be a socket
// address such as "1.1.1.1:53". DNSQuery is the domain queried when Mode is
// Socks5UDPProbeDNSRoundTrip.
type Socks5UDPProbeConfig struct {
	Mode          Socks5UDPProbeMode
	TimeoutMs     uint64
	DNSServerAddr string
	DNSServerHost string
	DNSQuery      string
}

// ffiSocks5UDPProbeConfig mirrors the C lk_socks5_udp_probe_config_t layout.
// The blank field reproduces the padding C inserts before the 8-byte aligned
// timeout_ms; on the supported 64-bit targets Go would add it anyway.
type ffiSocks5UDPProbeConfig struct {
	mode             int32
	_                int32
	timeoutMs        uint64
	dnsServerAddrPtr uintptr
	dnsServerAddrLen uintptr
	dnsServerHostPtr uintptr
	dnsServerHostLen uintptr
	dnsQueryPtr      uintptr
	dnsQueryLen      uintptr
}

// Socks5UDPProbeReport holds the outcome of a SOCKS5 UDP relay probe.
type Socks5UDPProbeReport struct {
	ptr  uintptr
	once sync.Once
}

func newSocks5UDPProbeReportFromPtr(ptr uintptr) *Socks5UDPProbeReport {
	if ptr == 0 {
		return nil
	}

	report := &Socks5UDPProbeReport{ptr: ptr}
	runtime.SetFinalizer(report, (*Socks5UDPProbeReport).Close)
	return report
}

// Phase reports how far the probe progressed.
func (r *Socks5UDPProbeReport) Phase() Socks5UDPProbePhase {
	if r == nil || r.ptr == 0 || ffi_lk_socks5_udp_probe_report_phase == nil {
		return Socks5UDPProbePhaseNotSocks5
	}
	return Socks5UDPProbePhase(ffi_lk_socks5_udp_probe_report_phase(r.ptr))
}

// Support summarizes the detected UDP relay support.
func (r *Socks5UDPProbeReport) Support() Socks5UDPProbeSupport {
	if r == nil || r.ptr == 0 || ffi_lk_socks5_udp_probe_report_support == nil {
		return Socks5UDPSupportFailed
	}
	return Socks5UDPProbeSupport(ffi_lk_socks5_udp_probe_report_support(r.ptr))
}

// ElapsedMs reports the probe duration in milliseconds.
func (r *Socks5UDPProbeReport) ElapsedMs() uint64 {
	if r == nil || r.ptr == 0 || ffi_lk_socks5_udp_probe_report_elapsed_ms == nil {
		return 0
	}
	return ffi_lk_socks5_udp_probe_report_elapsed_ms(r.ptr)
}

// JSON returns the full probe report encoded as JSON, or "".
func (r *Socks5UDPProbeReport) JSON() string {
	return r.readString(ffi_lk_socks5_udp_probe_report_json)
}

// ErrorMessage returns the failure detail captured by the probe, or "" when
// the probe did not record an error.
func (r *Socks5UDPProbeReport) ErrorMessage() string {
	return r.readString(ffi_lk_socks5_udp_probe_report_error)
}

// Proxy returns the proxy URL the probe targeted, or "".
func (r *Socks5UDPProbeReport) Proxy() string {
	return r.readString(ffi_lk_socks5_udp_probe_report_proxy)
}

// RelayAddr returns the UDP relay address advertised by the proxy, or "".
func (r *Socks5UDPProbeReport) RelayAddr() string {
	return r.readString(ffi_lk_socks5_udp_probe_report_relay_addr)
}

func (r *Socks5UDPProbeReport) readString(fn func(report uintptr, outPtr uintptr, outLen uintptr) int32) string {
	defer runtime.KeepAlive(r)
	if r == nil || r.ptr == 0 || fn == nil {
		return ""
	}

	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	if fn(r.ptr, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1]))) != ffiStatusOK {
		return ""
	}
	return goStringN(unsafe.Pointer(cell[0]), cell[1])
}

// Close releases the report. Safe to call multiple times.
func (r *Socks5UDPProbeReport) Close() {
	if r == nil {
		return
	}

	r.once.Do(func() {
		ptr := r.ptr
		r.ptr = 0
		runtime.SetFinalizer(r, nil)
		if ptr != 0 && ffi_lk_socks5_udp_probe_report_free != nil {
			ffi_lk_socks5_udp_probe_report_free(ptr)
		}
	})
}

func newFFISocks5UDPProbeConfig(config Socks5UDPProbeConfig) (ffiSocks5UDPProbeConfig, [][]byte) {
	addrPtr, addrBuf := optionalStringToCString(config.DNSServerAddr)
	hostPtr, hostBuf := optionalStringToCString(config.DNSServerHost)
	queryPtr, queryBuf := optionalStringToCString(config.DNSQuery)

	cfg := ffiSocks5UDPProbeConfig{
		mode:             int32(config.Mode),
		timeoutMs:        config.TimeoutMs,
		dnsServerAddrPtr: addrPtr,
		dnsServerAddrLen: uintptr(len(config.DNSServerAddr)),
		dnsServerHostPtr: hostPtr,
		dnsServerHostLen: uintptr(len(config.DNSServerHost)),
		dnsQueryPtr:      queryPtr,
		dnsQueryLen:      uintptr(len(config.DNSQuery)),
	}
	return cfg, [][]byte{addrBuf, hostBuf, queryBuf}
}

// Socks5UDPProbe probes whether the given SOCKS5 proxy supports UDP ASSOCIATE
// (and, in Socks5UDPProbeDNSRoundTrip mode, a relayed DNS round trip).
//
// Requires a QUIC/H3-capable lkrequest library; otherwise it returns an
// "unsupported" error. The returned report must be closed by the caller.
func (c *Client) Socks5UDPProbe(proxy string, config Socks5UDPProbeConfig) (*Socks5UDPProbeReport, error) {
	cp := c.loadPtr()
	if cp == 0 {
		return nil, ErrNilHandle
	}
	if ffi_lk_socks5_udp_probe == nil {
		return nil, unsupportedError("socks5 udp probe")
	}

	cfg, bufs := newFFISocks5UDPProbeConfig(config)
	proxyPtr, proxyBuf := stringToCString(proxy)

	// Pin cfg on the heap so callback-driven stack movement during the probe
	// cannot invalidate the address passed to FFI.
	cfgHeap, unpinCfg := ffiPinnedCopy(cfg)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_socks5_udp_probe(
		cp,
		proxyPtr,
		uintptr(len(proxy)),
		uintptr(unsafe.Pointer(cfgHeap)),
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outReport, outErr := cell[0], cell[1]
	unpinCfg()
	runtime.KeepAlive(bufs)
	runtime.KeepAlive(proxyBuf)
	runtime.KeepAlive(c)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "socks5 udp probe"); err != nil {
		return nil, err
	}
	if outReport == 0 {
		return nil, nilHandleError("socks5 udp probe")
	}

	return newSocks5UDPProbeReportFromPtr(outReport), nil
}

// Socks5UDPProbeAsync is the context-aware variant of Socks5UDPProbe. Cancelling
// ctx cancels the underlying operation.
func (c *Client) Socks5UDPProbeAsync(ctx context.Context, proxy string, config Socks5UDPProbeConfig) (*Socks5UDPProbeReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cp := c.loadPtr()
	if cp == 0 {
		return nil, ErrNilHandle
	}
	if ffi_lk_socks5_udp_probe_async == nil || ffi_lk_op_take_socks5_udp_probe_report == nil {
		return nil, unsupportedError("socks5 udp probe async")
	}

	cfg, bufs := newFFISocks5UDPProbeConfig(config)
	proxyPtr, proxyBuf := stringToCString(proxy)

	// Match the synchronous path: pin cfg on the heap and keep out parameters
	// in a heap cell.
	cfgHeap, unpinCfg := ffiPinnedCopy(cfg)
	cell := acquireFFIOutCell()
	defer ffiOutCellPool.Put(cell)
	status := ffi_lk_socks5_udp_probe_async(
		cp,
		proxyPtr,
		uintptr(len(proxy)),
		uintptr(unsafe.Pointer(cfgHeap)),
		uintptr(unsafe.Pointer(&cell[0])),
		uintptr(unsafe.Pointer(&cell[1])),
	)
	outOp, outErr := cell[0], cell[1]
	unpinCfg()
	runtime.KeepAlive(bufs)
	runtime.KeepAlive(proxyBuf)
	runtime.KeepAlive(c)

	if err := extractError(outErr); err != nil {
		return nil, err
	}
	if err := statusError(status, "socks5 udp probe async"); err != nil {
		return nil, err
	}
	if outOp == 0 {
		return nil, nilHandleError("socks5 udp probe async")
	}
	defer ffi_lk_op_free(outOp)

	state, err := waitForOp(ctx, outOp)
	if err != nil {
		return nil, err
	}

	return takeSocks5ReportFromOp(outOp, state)
}

func takeSocks5ReportFromOp(op uintptr, state OpState) (*Socks5UDPProbeReport, error) {
	switch state {
	case OpCompletedOK:
		cell := acquireFFIOutCell()
		defer ffiOutCellPool.Put(cell)
		status := ffi_lk_op_take_socks5_udp_probe_report(op, uintptr(unsafe.Pointer(&cell[0])), uintptr(unsafe.Pointer(&cell[1])))
		outReport, outErr := cell[0], cell[1]
		if err := extractError(outErr); err != nil {
			return nil, err
		}
		if err := statusError(status, "op take socks5 udp probe report"); err != nil {
			return nil, err
		}
		if outReport == 0 {
			return nil, nilHandleError("op take socks5 udp probe report")
		}
		return newSocks5UDPProbeReportFromPtr(outReport), nil
	case OpCompletedErr:
		return nil, takeOperationError(op, "socks5 udp probe")
	case OpCancelled:
		return nil, context.Canceled
	case OpConsumed:
		return nil, fmt.Errorf("lk: op already consumed")
	default:
		return nil, fmt.Errorf("lk: unexpected op state %s", state)
	}
}
