package fontmeta

import (
	"os"
	"path/filepath"
	"testing"

	tfont "github.com/tdewolff/font"
)

func TestFormatFromPath(t *testing.T) {
	cases := map[string]string{
		"demo.ttf":   "TTF",
		"demo.OTF":   "OTF",
		"demo.ttc":   "TTC",
		"demo.woff2": "WOFF2",
		"demo.pfb":   "TYPE1",
		"demo.fon":   "FON",
	}
	for path, want := range cases {
		got, ok := FormatFromPath(path)
		if !ok {
			t.Fatalf("expected %s to be supported", path)
		}
		if got != want {
			t.Fatalf("FormatFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestParseSampleFontWhenAvailable(t *testing.T) {
	path := `D:\MyFonts\FontExpertlvse\App\FontExpert\Font Samples\Aria.ttf`
	if _, err := os.Stat(path); err != nil {
		t.Skip("sample FontExpert font is not available")
	}
	parsed, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.File.Format != "TTF" {
		t.Fatalf("format = %s, want TTF", parsed.File.Format)
	}
	if len(parsed.Faces) == 0 {
		t.Fatal("expected at least one face")
	}
	if parsed.Faces[0].Family == "" {
		t.Fatal("expected parsed family name")
	}
}

func TestPreviewBytesSubsetsSampleText(t *testing.T) {
	path := bundledPreviewTestFont(t)
	preview, mime, stats, err := PreviewBytes(path, 0, "AaBbCc 0123456789")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "font/otf" {
		t.Fatalf("mime = %s, want font/otf", mime)
	}
	if len(preview) == 0 {
		t.Fatal("expected preview bytes")
	}
	if _, err := tfont.ParseSFNT(preview, 0); err != nil {
		t.Fatalf("subset preview is not parseable: %v", err)
	}
	if stats.Fallback {
		t.Fatalf("expected subset, got fallback: %s", stats.FallbackReason)
	}
	if stats.GlyphCount <= 1 {
		t.Fatalf("glyph count = %d, want more than .notdef", stats.GlyphCount)
	}
	if stats.SubsetBytes <= 0 || stats.FullBytes <= 0 {
		t.Fatalf("expected byte stats, got subset=%d full=%d", stats.SubsetBytes, stats.FullBytes)
	}
	if stats.SubsetBytes >= stats.FullBytes {
		t.Fatalf("subset bytes = %d, want less than full bytes %d", stats.SubsetBytes, stats.FullBytes)
	}
}

func TestPreviewBytesSkipsMissingRunes(t *testing.T) {
	path := bundledPreviewTestFont(t)
	preview, _, stats, err := PreviewBytes(path, 0, "A\U0010FFFF")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview) == 0 {
		t.Fatal("expected preview bytes")
	}
	if stats.MissingRuneCount == 0 {
		t.Fatal("expected missing rune count to be recorded")
	}
}

func bundledPreviewTestFont(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "frontend", "src", "assets", "fonts", "nunito-v16-latin-regular.woff2")
	if _, err := os.Stat(path); err != nil {
		t.Skip("bundled preview test font is not available")
	}
	return path
}
