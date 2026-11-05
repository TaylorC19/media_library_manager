package library

import (
	"strings"
	"testing"
)

func TestManualMetadataValidationAndCreatorNames(t *testing.T) {
	for _, url := range []string{"https://example.com/cover.jpg", " http://localhost:8080/cover.png "} {
		if len(validateManualMetadata(url, "Description", "Name")) != 0 {
			t.Errorf("valid cover rejected: %s", url)
		}
	}
	for _, url := range []string{"javascript:alert(1)", "data:image/png;base64,AAA", "file:///tmp/cover.png", "https://name:secret@example.com/cover.jpg", "/relative.jpg"} {
		if len(validateManualMetadata(url, "", "")) == 0 {
			t.Errorf("unsafe/invalid cover accepted: %s", url)
		}
	}
	if len(validateManualMetadata("", strings.Repeat("a", 10001), "")) == 0 || len(validateManualMetadata("", "", strings.Repeat("a", 5001))) == 0 {
		t.Error("metadata limits not enforced")
	}
	names := parseCreators(" First Creator\r\nFamily, Given\nfirst creator\n\n Second Creator ")
	if len(names) != 3 || names[1] != "Family, Given" || names[2] != "Second Creator" {
		t.Fatalf("creator names were split/lost: %v", names)
	}
}
