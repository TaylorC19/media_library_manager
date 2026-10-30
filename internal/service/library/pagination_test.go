package library

import "testing"

func TestOutOfRangeLibraryPageClampsToActualLastRows(t *testing.T) {
	for _, page := range []int{2, 3, 999} {
		start, end := pageBounds(15, page, 10)
		if start != 10 || end != 15 {
			t.Errorf("page %d: rows %d-%d, want 10-15", page, start, end)
		}
	}
}
