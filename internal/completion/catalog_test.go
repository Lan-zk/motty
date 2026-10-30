package completion

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	catalog := testCatalog(t)
	encoded := catalog.Encode()
	if !bytes.HasPrefix(encoded, []byte("motty-completion\t1\n")) {
		t.Fatalf("unexpected header: %q", encoded[:20])
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded.RootCount() != catalog.RootCount() || decoded.NodeCount() != catalog.NodeCount() || decoded.ValueCount() != catalog.ValueCount() {
		t.Fatalf("counts differ: %+v vs %+v", decoded, catalog)
	}
	if !bytes.Equal(decoded.Encode(), encoded) {
		t.Fatal("re-encoding a decoded catalog changed its bytes")
	}
	node := decoded.Lookup([]string{"git", "commit"})
	if node == nil {
		t.Fatal("git commit is missing after decoding")
	}
	if node.Parent() == nil || node.Parent().Token != "git" {
		t.Fatalf("parent link is wrong: %+v", node.Parent())
	}
	flags := decoded.VisibleFlags(node)
	if len(flags) != 6 {
		t.Fatalf("visible flags = %d, want 6", len(flags))
	}
	format, ok := decoded.FindFlag(node, "--format")
	if !ok || !format.Value || len(format.Values) != 2 {
		t.Fatalf("--format = %+v, %v", format, ok)
	}
}

func TestDecodeRejectsCorruptCatalogs(t *testing.T) {
	encoded := testCatalog(t).Encode()
	corruptions := map[string]func([]byte) []byte{
		"truncated": func(data []byte) []byte { return data[:len(data)-20] },
		"bad digest": func(data []byte) []byte {
			copy := append([]byte(nil), data...)
			copy[len(copy)-1] ^= 0xff
			return copy
		},
		"wrong count": func(data []byte) []byte {
			return bytes.Replace(data, []byte("nodes\t"), []byte("nodes\t9"), 1)
		},
	}
	for name, corrupt := range corruptions {
		if _, err := Decode(corrupt(encoded)); err == nil {
			t.Errorf("%s: Decode accepted a corrupt catalog", name)
		}
	}
	if _, err := Decode(nil); err == nil {
		t.Error("Decode(nil) succeeded")
	}
}

func TestNewCatalogRejectsDuplicates(t *testing.T) {
	roots := []*Node{{Token: "a"}, {Token: "a"}}
	if _, err := NewCatalog("s", "r", "c", "MIT", roots); err == nil {
		t.Fatal("duplicate roots were accepted")
	}
	if _, err := NewCatalog("s", "r", "c", "MIT", nil); err == nil {
		t.Fatal("empty catalog was accepted")
	}
}
