package lkrequest

import (
	"os"
	"os/exec"
	"sync"
	"testing"
)

func TestInitLogCallbackSubprocess(t *testing.T) {
	if os.Getenv("LKREQUEST_TEST_LOG_CALLBACK_SUBPROCESS") == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestInitLogCallbackSubprocess$")
		cmd.Env = append(os.Environ(), "LKREQUEST_TEST_LOG_CALLBACK_SUBPROCESS=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("log callback subprocess failed: %v\n%s", err, string(output))
		}
		return
	}

	var (
		mu          sync.Mutex
		entryCount  int
		targetSeen  bool
		messageSeen bool
	)

	err := InitLogCallback(func(level LogLevel, target, message string) {
		mu.Lock()
		defer mu.Unlock()

		entryCount++
		if target != "" {
			targetSeen = true
		}
		if message != "" {
			messageSeen = true
		}
	}, LogLevelTrace)
	if err != nil {
		t.Fatalf("InitLogCallback() error = %v", err)
	}

	server := newLocalFeatureServer(t)
	client, session := newLocalSession(t)
	t.Cleanup(client.Close)
	t.Cleanup(session.Close)

	resp := mustSendRequest(t, mustNewRequest(t, session, "GET", server.URL+"/echo"))
	cleanupResponse(t, resp)

	mu.Lock()
	defer mu.Unlock()

	if entryCount == 0 {
		t.Fatal("log callback did not receive any entries")
	}
	if !targetSeen {
		t.Fatal("log callback did not receive a non-empty target")
	}
	if !messageSeen {
		t.Fatal("log callback did not receive a non-empty message")
	}
}
