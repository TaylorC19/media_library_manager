package httpx

// Page keeps provider totals with a batch rather than treating the batch as the
// entire result set. Total is the number of results accessible through the API.
type Page[T any] struct {
	Items []T
	Total int
}

func (p Page[T]) IsEmpty() bool { return len(p.Items) == 0 }
