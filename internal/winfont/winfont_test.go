package winfont

import "testing"

func TestRegistryValueName(t *testing.T) {
	if got := RegistryValueName("Demo Regular", "TTF"); got != "Demo Regular (TrueType)" {
		t.Fatalf("TTF value name = %q", got)
	}
	if got := RegistryValueName("Demo", "OTF"); got != "Demo (OpenType)" {
		t.Fatalf("OTF value name = %q", got)
	}
	if got := RegistryValueName("Bitmap", "FON"); got != "Bitmap" {
		t.Fatalf("FON value name = %q", got)
	}
}
