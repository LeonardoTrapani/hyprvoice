package bus

import "testing"

func TestSockPath(t *testing.T) {
	if _, err := SockPath(); err != nil {
		t.Fatalf("SockPath: %v", err)
	}
}
