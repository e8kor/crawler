package function

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestHandleReturnsCorrectResponse(t *testing.T) {
	entries := CollectEntries("https://www.otodom.pl/wynajem/lokal/krakow/?search%5Bcity_id%5D=38")

	if len(entries) == 0 {
		t.Fatal("Expected entries to be non empty")
	} else {
		fmt.Println("response:", entries)
	}
}

func TestHandleExtractNumber(t *testing.T) {
	raw := []string{
		"40 zł/m²",
		"39 m²",
		"1 100 zł                                                        /mc",
	}
	for _, item := range raw {
		num, err := strconv.Atoi(ExtractNumber(item))
		if err != nil {
			t.Fatal("cannot extract number from string", item)
		} else {
			fmt.Println("parsed:", num)
		}
	}
}

func TestHandleTakeChractersBefore(t *testing.T) {
	var (
		raw       = "https://www.otodom.pl/pl/oferta/atrakcyjny-lokal-mieszkanie-po-remoncie-kazimierz-ID43fpP.html#dac7588e86"
		predicate = ".html"
	)
	prepared := TakeChractersBefore(raw, predicate)
	if strings.Contains(prepared, predicate) {
		t.Fatal("cannot extract number from string", raw)
	}
}
