package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ExportDatabase reads all backup collections and returns a gzip-compressed JSON archive.
func ExportDatabase(ctx context.Context, db *mongo.Database) ([]byte, error) {
	collections := make(map[string][]string, len(CollectionNames))
	for _, name := range CollectionNames {
		docs, err := exportCollection(ctx, db.Collection(name))
		if err != nil {
			return nil, fmt.Errorf("export collection %q: %w", name, err)
		}
		collections[name] = docs
	}

	payload := File{
		Manifest: Manifest{
			Format:     FormatName,
			Version:    FormatVersion,
			ExportedAt: time.Now().UTC(),
			Database:   db.Name(),
		},
		Collections: collections,
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		_ = gz.Close()
		return nil, fmt.Errorf("encode backup: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("close gzip: %w", err)
	}
	return buf.Bytes(), nil
}

func exportCollection(ctx context.Context, coll *mongo.Collection) ([]string, error) {
	cur, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []string
	for cur.Next(ctx) {
		var raw bson.Raw
		if err := cur.Decode(&raw); err != nil {
			return nil, err
		}
		ext, err := bson.MarshalExtJSON(raw, false, false)
		if err != nil {
			return nil, err
		}
		out = append(out, string(ext))
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}
