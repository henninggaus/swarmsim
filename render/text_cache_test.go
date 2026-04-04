package render

import "testing"

func TestClearTextCacheEmpty(t *testing.T) {
	// Cache should be empty at start
	ClearTextCache()
	if len(textCache) != 0 {
		t.Errorf("expected empty cache, got %d entries", len(textCache))
	}
}

func TestTextCacheInsertAndClear(t *testing.T) {
	// Manually insert a fake entry
	textCache["test_key"] = &textCacheEntry{img: nil, lastUsed: 0}
	if len(textCache) != 1 {
		t.Fatal("expected 1 entry after insert")
	}
	ClearTextCache()
	if len(textCache) != 0 {
		t.Errorf("expected empty cache after clear, got %d", len(textCache))
	}
}
