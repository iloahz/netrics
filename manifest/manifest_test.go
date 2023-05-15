package manifest

import (
	"encoding/json"
	"os"
	"testing"
)

func TestUpdateManifest(t *testing.T) {
	buf, err := os.ReadFile("input.json")
	if err != nil {
		t.Fatal(err)
	}
	var input Input
	err = json.Unmarshal(buf, &input)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := BuildManifest(&input)
	if err != nil {
		t.Fatal(err)
	}
	buf, err = json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile("manifest.json", buf, 0644)
	if err != nil {
		t.Fatal(err)
	}
}
