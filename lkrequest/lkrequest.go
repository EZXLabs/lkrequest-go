// Package lk provides Go bindings for the lkrequest HTTP client library.
//
// The library supports two binding engines selectable via build tags:
//   - Default (purego): No C compiler required, uses runtime dynamic loading
//   - CGo (-tags lkcgo): Static linking, requires C compiler, best performance
//
// Quick start:
//
//	resp, err := lk.Get("https://example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer resp.Close()
//	fmt.Println(resp.String())
package lkrequest

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

var (
	defaultOnce    sync.Once
	defaultClient  *Client
	defaultSession *Session
	defaultErr     error

	quicH3FeatureOnce      sync.Once
	quicH3FeatureSupported bool
)

func ABIVersion() uint32 {
	return ffi_lk_abi_version()
}

func LibraryVersion() string {
	return goCString(ffi_lk_library_version())
}

func FeatureSupported(name string) bool {
	namePtr, nameBuf := stringToCString(name)
	supported := ffiBool(ffi_lk_feature_supported(namePtr))
	runtime.KeepAlive(nameBuf)
	if supported {
		return true
	}

	switch strings.ToLower(strings.TrimSpace(name)) {
	case "quic-h3":
		return detectQUICH3FeatureSupport()
	default:
		return false
	}
}

type fingerprintFeatureInfo struct {
	Quic json.RawMessage `json:"quic"`
}

// Older lkrequest-ffi builds expose QUIC/H3 APIs but may not advertise them
// through lk_feature_supported("quic-h3"). Fall back to the fingerprint JSON.
func detectQUICH3FeatureSupport() bool {
	quicH3FeatureOnce.Do(func() {
		client, err := NewClient("chrome_144")
		if err != nil {
			client, err = NewDefaultClient()
		}
		if err != nil || client == nil {
			return
		}
		defer client.Close()

		info, err := client.FingerprintInfoJSON()
		if err != nil {
			return
		}

		var payload fingerprintFeatureInfo
		if err := json.Unmarshal([]byte(info), &payload); err != nil {
			return
		}

		raw := strings.TrimSpace(string(payload.Quic))
		quicH3FeatureSupported = raw != "" && raw != "null"
	})

	return quicH3FeatureSupported
}

func getDefault() (*Session, error) {
	defaultOnce.Do(func() {
		var err error
		defaultClient, err = NewDefaultClient()
		if err != nil {
			defaultErr = err
			return
		}

		defaultSession, defaultErr = NewSession(defaultClient)
	})

	return defaultSession, defaultErr
}

func Get(url string) (*Response, error) {
	session, err := getDefault()
	if err != nil {
		return nil, fmt.Errorf("lk: default session init failed: %w", err)
	}

	req, err := NewRequest(session, "GET", url)
	if err != nil {
		return nil, err
	}

	return req.Send()
}

func PostJSON(url, jsonBody string) (*Response, error) {
	session, err := getDefault()
	if err != nil {
		return nil, fmt.Errorf("lk: default session init failed: %w", err)
	}

	req, err := NewRequest(session, "POST", url)
	if err != nil {
		return nil, err
	}

	return req.SetJSONBody(jsonBody).Send()
}
