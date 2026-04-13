package library

const (
	SourceManual   = "manual"
	SourceProvider = "provider"
)

const (
	BucketCatalog  = "catalog"
	BucketWishlist = "wishlist"
)

var (
	Buckets     = []string{BucketCatalog, BucketWishlist}
	MediaTypes  = []string{"movie", "tv", "album", "book", "game"}
	Formats = []string{"blu_ray", "dvd", "vhs", "cd", "vinyl", "cassette", "hardcover", "paperback", "switch", "ps5", "xbox", "digital", "other"}
)

func IsBucket(s string) bool {
	for _, b := range Buckets {
		if b == s {
			return true
		}
	}
	return false
}

func IsMediaType(s string) bool {
	for _, m := range MediaTypes {
		if m == s {
			return true
		}
	}
	return false
}

func IsFormat(s string) bool {
	if s == "" {
		return true
	}
	for _, f := range Formats {
		if f == s {
			return true
		}
	}
	return false
}
