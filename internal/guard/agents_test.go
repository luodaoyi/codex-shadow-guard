package guard

import "testing"

func TestReplaceAndRemoveManagedBlockPreservesUserContent(t *testing.T) {
	original := "# Existing rules\n\nKeep this text.\n"
	installed, changed, err := replaceManagedBlock(original, AgentsBlock)
	if err != nil || !changed {
		t.Fatalf("install: changed=%v err=%v", changed, err)
	}
	if got := count(installed, AgentsBegin); got != 1 {
		t.Fatalf("begin markers = %d", got)
	}
	removed, changed, err := removeManagedBlock(installed)
	if err != nil || !changed {
		t.Fatalf("remove: changed=%v err=%v", changed, err)
	}
	if removed != original {
		t.Fatalf("user content changed: %q", removed)
	}
}

func TestRejectsMalformedManagedBlock(t *testing.T) {
	_, _, err := removeManagedBlock(AgentsEnd)
	if err == nil {
		t.Fatal("expected malformed block error")
	}
}

func count(value, needle string) int {
	result := 0
	for offset := 0; ; {
		index := offset + indexOf(value[offset:], needle)
		if index < offset {
			return result
		}
		result++
		offset = index + len(needle)
	}
}

func indexOf(value, needle string) int {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
