package manifest

import (
	"testing"
)

func TestSummarizeWebsite(t *testing.T) {
	res, err := SummarizeWebsite("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}
