package library

const DefaultSort = "recent"

var SortOrders = []string{"recent", "title_asc", "title_desc", "year_desc", "year_asc"}

func IsLibrarySort(value string) bool {
	if value == "" {
		return true
	}
	for _, order := range SortOrders {
		if value == order {
			return true
		}
	}
	return false
}
