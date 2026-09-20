package quark

import (
	"strings"
	"testing"
)

// fileHashes must return the real SHA-1: Quark's upload callback validates the
// object digest (content_hash) and rejects the completion with HTTP 203
// (CallbackFailed) when the value is wrong (previously SHA-1 was computed as
// MD5, i.e. both digests were identical).
func TestFileHashes(t *testing.T) {
	md5hex, sha1hex, err := fileHashes(strings.NewReader("abc"))
	if err != nil {
		t.Fatalf("fileHashes: %v", err)
	}
	if want := "900150983cd24fb0d6963f7d28e17f72"; md5hex != want {
		t.Errorf("md5 = %s, want %s", md5hex, want)
	}
	if want := "a9993e364706816aba3e25717850c26c9cd0d89d"; sha1hex != want {
		t.Errorf("sha1 = %s, want %s", sha1hex, want)
	}
	if md5hex == sha1hex {
		t.Error("md5 and sha1 digests must differ")
	}
}
