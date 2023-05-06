package manifest

import "testing"

func TestUpdateManifest(t *testing.T) {
	err := UpdateManifest("websites.json")
	if err != nil {
		t.Fatal(err)
	}
}
