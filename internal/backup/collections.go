package backup

// CollectionNames lists MongoDB collections included in full-database export/import.
var CollectionNames = []string{
	"users",
	"sessions",
	"library_entries",
	"media_records",
	"provider_cache",
	"scan_logs",
}
