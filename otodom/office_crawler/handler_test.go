package function

import (
	"fmt"
	"testing"
)

func TestHandleReturnsCorrectResponse(t *testing.T) {
	entries, _ := CollectEntries("https://www.otodom.pl/wynajem/lokal/krakow/?search%5Bcity_id%5D=38")

	if len(entries) == 0 {
		t.Fatal("Expected entries to be non empty")
	} else {
		fmt.Println("response:", entries)
	}
}
