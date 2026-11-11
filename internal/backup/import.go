package backup

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// ImportOptions controls database restore behavior.
type ImportOptions struct {
	// Atomic requires a replica set and rolls back every collection on failure.
	Atomic bool
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

	// Parse and validate the entire archive before the first database mutation.
	staged := make(map[string][]any, len(CollectionNames))
	for _, name := range CollectionNames {
		extDocs, ok := file.Collections[name]
		if !ok {
			return nil, fmt.Errorf("backup missing collection %q", name)
		}
		for _, ext := range extDocs {
			var doc bson.M
			if err := bson.UnmarshalExtJSON([]byte(ext), false, &doc); err != nil {
				return nil, fmt.Errorf("invalid document in collection %q", name)
			}
			encoded, err := bson.Marshal(doc)
			if err != nil || len(encoded) > 16*1024*1024 {
				return nil, fmt.Errorf("invalid document size in collection %q", name)
			}
			staged[name] = append(staged[name], doc)
		}
	}
	result := &ImportResult{Counts: make(map[string]int, len(CollectionNames))}
	restore := func(ctx context.Context) error {
		for _, name := range CollectionNames {
			n, err := replaceCollection(ctx, db.Collection(name), staged[name])
			if err != nil {
				return fmt.Errorf("import collection %q: %w", name, err)
			}
			result.Counts[name] = n
		}
		return nil
	}
	if opts.Atomic {
		session, err := db.Client().StartSession()
		if err != nil {
			return nil, err
		}
		defer session.EndSession(ctx)
		_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) { return nil, restore(sc) }, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
		if err != nil {
			return nil, err
		}
	} else if err := restore(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeBackupFile(r io.Reader) (*File, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("read gzip backup: %w", err)
	}
	defer gz.Close()

	const maxExpandedBytes = 128 * 1024 * 1024
	raw, err := io.ReadAll(io.LimitReader(gz, maxExpandedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read backup: %w", err)
	}
	if len(raw) > maxExpandedBytes {
		return nil, fmt.Errorf("expanded backup exceeds size limit")
	}
	var file File
	if err := json.Unmarshal(raw, &file); err != nil {
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

func replaceCollection(ctx context.Context, coll *mongo.Collection, docs []any) (int, error) {
	if _, err := coll.DeleteMany(ctx, bson.M{}); err != nil {
		return 0, err
	}
	if len(docs) == 0 {
		return 0, nil
	}

	res, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return 0, err
	}
	return len(res.InsertedIDs), nil
}
