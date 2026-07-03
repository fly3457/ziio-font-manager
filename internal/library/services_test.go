package library

import (
	"strings"
	"testing"
)

func TestPreviewCacheKeyIncludesSampleHash(t *testing.T) {
	fontHash := "abcdef0123456789"
	first := previewCacheKey(fontHash, "", 0, previewSampleHash("AaBbCc"))
	same := previewCacheKey(fontHash, "", 0, previewSampleHash("AaBbCc"))
	other := previewCacheKey(fontHash, "", 0, previewSampleHash("012345"))

	if !strings.HasPrefix(first, "v2-") {
		t.Fatalf("cache key = %q, want v2 prefix", first)
	}
	if first != same {
		t.Fatalf("same sample text produced different cache keys: %q vs %q", first, same)
	}
	if first == other {
		t.Fatalf("different sample text produced same cache key: %q", first)
	}
}
