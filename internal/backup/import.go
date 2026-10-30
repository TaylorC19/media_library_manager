package backup

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ImportOptions controls database restore behavior.
type ImportOptions struct {
	// RequireDatabaseMatch rejects backups whose manifest.database differs from the target db name.
	RequireDatabaseMatch bool
}

// ImportResult summarizes a restore operation.
type ImportResult struct {
	Counts map[string]int
}

// ImportDatabase replaces all documents in backup collections from a gzip-compressed JSON archive.
func ImportDatabase(ctx context.Context, db *mongo.Database, r io.Reader, opts ImportOptions) (*ImportResult, error) {
	file, err := decodeBackupFile(r)
	if err != nil {
		return nil, err
	}
	if err := validateManifest(file.Manifest, db.Name(), opts); err != nil {
		return nil, err
	}

	result := &ImportResult{Counts: make(map[string]int, len(CollectionNames))}
	for _, name := range CollectionNames {
		docs, ok := file.Collections[name]
		if !ok {
			return nil, fmt.Errorf("backup missing collection %q", name)
		}
		n, err := replaceCollection(ctx, db.Collection(name), docs)
		if err != nil {
			return nil, fmt.Errorf("import collection %q: %w", name, err)
		}
		result.Counts[name] = n
	}
	return result, nil
}

func decodeBackupFile(r io.Reader) (*File, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("read gzip backup: %w", err)
	}
	defer gz.Close()

	var file File
	dec := json.NewDecoder(gz)
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("decode backup json: %w", err)
	}
	return &file, nil
}

func validateManifest(m Manifest, targetDB string, opts ImportOptions) error {
	if m.Format != FormatName {
		return fmt.Errorf("unsupported backup format %q", m.Format)
	}
	if m.Version != FormatVersion {
		return fmt.Errorf("unsupported backup version %d (want %d)", m.Version, FormatVersion)
	}
	if opts.RequireDatabaseMatch && m.Database != targetDB {
		return fmt.Errorf("backup database %q does not match target %q", m.Database, targetDB)
	}
	return nil
}

func replaceCollection(ctx context.Context, coll *mongo.Collection, extDocs []string) (int, error) {
	if _, err := coll.DeleteMany(ctx, bson.M{}); err != nil {
		return 0, err
	}
	if len(extDocs) == 0 {
		return 0, nil
	}

	docs := make([]any, 0, len(extDocs))
	for _, ext := range extDocs {
		var doc bson.M
		if err := bson.UnmarshalExtJSON([]byte(ext), false, &doc); err != nil {
			return 0, fmt.Errorf("unmarshal document: %w", err)
		}
		docs = append(docs, doc)
	}

	res, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return 0, err
	}
	return len(res.InsertedIDs), nil
}

