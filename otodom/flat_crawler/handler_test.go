package function

import (
	"fmt"
	"testing"
)

func TestHandleReturnsCorrectResponse(t *testing.T) {
	entries := CollectEntries("https://www.otodom.pl/wynajem/mieszkanie/krakow/?search%5Bcity_id%5D=38")

	if len(entries) == 0 {
		t.Fatal("Expected entries to be non empty")
	} else {
		fmt.Println("response:", entries)
	}
}
