package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	domainlib "media_library_manager/internal/domain/library"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MediaRecordsRepository struct {
	coll *mongo.Collection
}

func NewMediaRecordsRepository(db *mongo.Database) *MediaRecordsRepository {
	return &MediaRecordsRepository{coll: db.Collection("media_records")}
}

func (r *MediaRecordsRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "mediaType", Value: 1}, {Key: "title", Value: 1}, {Key: "year", Value: 1}}},
		{Keys: bson.D{{Key: "mediaType", Value: 1}, {Key: "sortTitle", Value: 1}, {Key: "year", Value: 1}}},
		{Keys: bson.D{{Key: "barcodeCandidates", Value: 1}}},
		{Keys: bson.D{{Key: "source", Value: 1}}},
		{Keys: bson.D{{Key: "providerRefs.tmdb.id", Value: 1}, {Key: "providerRefs.tmdb.kind", Value: 1}}},
		{Keys: bson.D{{Key: "providerRefs.musicbrainz.releaseId", Value: 1}}},
		{Keys: bson.D{{Key: "providerRefs.openLibrary.workKey", Value: 1}}},
		{Keys: bson.D{{Key: "providerRefs.rawg.id", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("create media_records indexes: %w", err)
	}
	return nil
}

func (r *MediaRecordsRepository) Insert(ctx context.Context, rec *domainlib.MediaRecord) error {
	now := time.Now().UTC()
	rec.CreatedAt = now
	rec.UpdatedAt = now
	if rec.Details == nil {
		rec.Details = bson.M{}
	}
	if rec.ProviderRefs == nil {
		rec.ProviderRefs = bson.M{}
	}

	res, err := r.coll.InsertOne(ctx, rec)
	if err != nil {
		return fmt.Errorf("insert media record: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		rec.ID = oid
	}
	return nil
}

func (r *MediaRecordsRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domainlib.MediaRecord, error) {
	var rec domainlib.MediaRecord
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&rec)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find media record: %w", err)
	}
	return &rec, nil
}

func (r *MediaRecordsRepository) FindByProviderImportRef(ctx context.Context, provider, externalID, tmdbKind string) (*domainlib.MediaRecord, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return nil, nil
	}

	query := bson.M{}
	switch strings.TrimSpace(provider) {
	case "tmdb":
		query["providerRefs.tmdb.id"] = externalID
		query["providerRefs.tmdb.kind"] = strings.TrimSpace(tmdbKind)
	case "musicbrainz":
		query["providerRefs.musicbrainz.releaseId"] = externalID
	case "open_library":
		query["providerRefs.openLibrary.workKey"] = externalID
	case "rawg":
		query["providerRefs.rawg.id"] = externalID
	default:
		return nil, nil
	}

	var rec domainlib.MediaRecord
	err := r.coll.FindOne(ctx, query).Decode(&rec)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find media record by provider ref: %w", err)
	}
	return &rec, nil
}

// FindByBarcodeCandidate finds media records whose barcodeCandidates contain the given normalized barcode.
// If mediaType is non-nil and non-empty, results are restricted to that media type.
func (r *MediaRecordsRepository) FindByBarcodeCandidate(ctx context.Context, barcode string, mediaType *string) ([]domainlib.MediaRecord, error) {
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		return nil, nil
	}
	q := bson.M{"barcodeCandidates": barcode}
	if mediaType != nil && strings.TrimSpace(*mediaType) != "" {
		q["mediaType"] = strings.TrimSpace(*mediaType)
	}
	cur, err := r.coll.Find(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("find media records by barcode candidate: %w", err)
	}
	defer cur.Close(ctx)

	var out []domainlib.MediaRecord
	for cur.Next(ctx) {
		var rec domainlib.MediaRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("decode media record: %w", err)
		}
		out = append(out, rec)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate media records by barcode: %w", err)
	}
	return out, nil
}

func (r *MediaRecordsRepository) FindByAnyBarcodeCandidates(ctx context.Context, mediaType string, barcodes []string) ([]domainlib.MediaRecord, error) {
	if len(barcodes) == 0 {
		return nil, nil
	}

	cur, err := r.coll.Find(ctx, bson.M{
		"mediaType":         mediaType,
		"barcodeCandidates": bson.M{"$in": barcodes},
	})
	if err != nil {
		return nil, fmt.Errorf("find media records by barcode candidates: %w", err)
	}
	defer cur.Close(ctx)

	var out []domainlib.MediaRecord
	for cur.Next(ctx) {
		var rec domainlib.MediaRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("decode media record: %w", err)
		}
		out = append(out, rec)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate media records by barcode candidates: %w", err)
	}
	return out, nil
}

func (r *MediaRecordsRepository) FindLooseTitleYear(ctx context.Context, mediaType, title string, year *int32) ([]domainlib.MediaRecord, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, nil
	}

	query := bson.M{
		"mediaType": mediaType,
		"title": bson.M{
			"$regex":   "^" + regexp.QuoteMeta(title) + "$",
			"$options": "i",
		},
	}
	if year != nil {
		query["year"] = *year
	}

	cur, err := r.coll.Find(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find media records by title/year: %w", err)
	}
	defer cur.Close(ctx)

	var out []domainlib.MediaRecord
	for cur.Next(ctx) {
		var rec domainlib.MediaRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("decode media record: %w", err)
		}
		out = append(out, rec)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate media records by title/year: %w", err)
	}
	return out, nil
}

func (r *MediaRecordsRepository) UpdateManualCore(ctx context.Context, id primitive.ObjectID, title string, sortTitle *string, mediaType string, year *int32) error {
	set := bson.M{
		"title":     title,
		"mediaType": mediaType,
		"updatedAt": time.Now().UTC(),
	}
	if sortTitle != nil {
		set["sortTitle"] = *sortTitle
	} else {
		set["sortTitle"] = nil
	}
	if year != nil {
		set["year"] = *year
	} else {
		set["year"] = nil
	}

	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id, "source": domainlib.SourceManual}, bson.M{"$set": set})
	if err != nil {
		return fmt.Errorf("update media record: %w", err)
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MediaRecordsRepository) UpdateProviderData(ctx context.Context, rec *domainlib.MediaRecord) error {
	if rec == nil {
		return mongo.ErrNilDocument
	}

	now := time.Now().UTC()
	rec.UpdatedAt = now
	set := bson.M{
		"source":            rec.Source,
		"mediaType":         rec.MediaType,
		"title":             rec.Title,
		"sortTitle":         rec.SortTitle,
		"releaseDate":       rec.ReleaseDate,
		"year":              rec.Year,
		"imageUrl":          rec.ImageURL,
		"summary":           rec.Summary,
		"providerRefs":      rec.ProviderRefs,
		"externalRatings":   rec.ExternalRatings,
		"barcodeCandidates": rec.BarcodeCandidates,
		"details":           rec.Details,
		"lastSyncedAt":      rec.LastSyncedAt,
		"updatedAt":         now,
	}

	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": rec.ID}, bson.M{"$set": set})
	if err != nil {
		return fmt.Errorf("update provider media record: %w", err)
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MediaRecordsRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete media record: %w", err)
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MediaRecordsRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) (map[primitive.ObjectID]*domainlib.MediaRecord, error) {
	if len(ids) == 0 {
		return map[primitive.ObjectID]*domainlib.MediaRecord{}, nil
	}
	cur, err := r.coll.Find(ctx, bson.M{"_id": bson.M{"$in": ids}}, options.Find().SetProjection(bson.M{
		"title": 1, "year": 1, "mediaType": 1, "source": 1, "summary": 1, "imageUrl": 1,
	}))
	if err != nil {
		return nil, fmt.Errorf("find media records by ids: %w", err)
	}
	defer cur.Close(ctx)

	out := make(map[primitive.ObjectID]*domainlib.MediaRecord)
	for cur.Next(ctx) {
		var rec domainlib.MediaRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, fmt.Errorf("decode media record: %w", err)
		}
		out[rec.ID] = &rec
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
