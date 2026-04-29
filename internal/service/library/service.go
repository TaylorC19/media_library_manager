package library

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("not found")
)

type Service struct {
	media   *repository.MediaRecordsRepository
	entries *repository.LibraryEntriesRepository
}

func NewService(media *repository.MediaRecordsRepository, entries *repository.LibraryEntriesRepository) *Service {
	return &Service{media: media, entries: entries}
}

type EntryWithMedia struct {
	Entry *domainlib.LibraryEntry
	Media *domainlib.MediaRecord
}

type ListBucketFilters struct {
	MediaType string
	Format    string
	Query     string
}

type ListBucketResult struct {
	Items      []EntryWithMedia
	TotalItems int
}

type ManualCreateForm struct {
	Bucket       string
	MediaType    string
	Title        string
	Year         *int
	Format       string
	Barcode      string
	PurchaseDate string
	Notes        string
	Tags         string
}

type ManualUpdateForm struct {
	MediaType    string
	Title        string
	Year         *int
	Bucket       string
	Format       string
	Barcode      string
	PurchaseDate string
	Notes        string
	Tags         string
}

func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	seen := map[string]struct{}{}
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

func optionalString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func validateCore(mediaType, bucket, format string) []string {
	var errs []string
	if !domainlib.IsMediaType(mediaType) {
		errs = append(errs, "library.errors.invalidMediaType")
	}
	if !domainlib.IsBucket(bucket) {
		errs = append(errs, "library.errors.invalidBucket")
	}
	if format != "" && !domainlib.IsFormat(format) {
		errs = append(errs, "library.errors.invalidFormat")
	}
	return errs
}

func (s *Service) DashboardStats(ctx context.Context, userID primitive.ObjectID) (catalogCount, wishlistCount int64, err error) {
	catalogCount, err = s.entries.CountByUserAndBucket(ctx, userID, domainlib.BucketCatalog)
	if err != nil {
		return 0, 0, err
	}
	wishlistCount, err = s.entries.CountByUserAndBucket(ctx, userID, domainlib.BucketWishlist)
	if err != nil {
		return 0, 0, err
	}
	return catalogCount, wishlistCount, nil
}

func (s *Service) ListBucket(ctx context.Context, userID primitive.ObjectID, bucket string, filters ListBucketFilters, page, pageSize int) (*ListBucketResult, error) {
	if !domainlib.IsBucket(bucket) {
		return nil, fmt.Errorf("bucket: %w", ErrValidation)
	}
	if filters.MediaType != "" && !domainlib.IsMediaType(filters.MediaType) {
		return nil, fmt.Errorf("media type: %w", ErrValidation)
	}
	if filters.Format != "" && !domainlib.IsFormat(filters.Format) {
		return nil, fmt.Errorf("format: %w", ErrValidation)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	entries, err := s.entries.ListByUserAndBucket(ctx, userID, bucket, repository.LibraryListFilter{
		MediaType: filters.MediaType,
		Format:    filters.Format,
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.attachMedia(ctx, entries)
	if err != nil {
		return nil, err
	}

	rows = filterRows(rows, filters.Query)
	totalItems := len(rows)
	start, end := pageBounds(totalItems, page, pageSize)
	return &ListBucketResult{
		Items:      rows[start:end],
		TotalItems: totalItems,
	}, nil
}

func (s *Service) attachMedia(ctx context.Context, entries []domainlib.LibraryEntry) ([]EntryWithMedia, error) {
	idSet := make(map[primitive.ObjectID]struct{})
	var ids []primitive.ObjectID
	for _, e := range entries {
		if _, ok := idSet[e.MediaRecordID]; ok {
			continue
		}
		idSet[e.MediaRecordID] = struct{}{}
		ids = append(ids, e.MediaRecordID)
	}
	mediaByID, err := s.media.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]EntryWithMedia, 0, len(entries))
	for i := range entries {
		m := mediaByID[entries[i].MediaRecordID]
		out = append(out, EntryWithMedia{Entry: &entries[i], Media: m})
	}
	return out, nil
}

func filterRows(rows []EntryWithMedia, query string) []EntryWithMedia {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return rows
	}

	filtered := make([]EntryWithMedia, 0, len(rows))
	for _, row := range rows {
		title := ""
		summary := ""
		if row.Media != nil {
			title = strings.ToLower(strings.TrimSpace(row.Media.Title))
			if row.Media.Summary != nil {
				summary = strings.ToLower(strings.TrimSpace(*row.Media.Summary))
			}
		}
		notes := strings.ToLower(strings.TrimSpace(stringValue(row.Entry.Notes)))
		if strings.Contains(title, query) || strings.Contains(summary, query) || strings.Contains(notes, query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func pageBounds(totalItems, page, pageSize int) (int, int) {
	if totalItems <= 0 {
		return 0, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= totalItems {
		return totalItems, totalItems
	}
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}
	return start, end
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Service) GetEntryDetail(ctx context.Context, userID, entryID primitive.ObjectID) (*EntryWithMedia, error) {
	entry, err := s.entries.FindByIDAndUser(ctx, entryID, userID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, ErrNotFound
	}
	media, err := s.media.FindByID(ctx, entry.MediaRecordID)
	if err != nil {
		return nil, err
	}
	if media == nil {
		return nil, ErrNotFound
	}
	return &EntryWithMedia{Entry: entry, Media: media}, nil
}

func (s *Service) CreateManual(ctx context.Context, userID primitive.ObjectID, f ManualCreateForm) (primitive.ObjectID, []string, error) {
	var errs []string
	if strings.TrimSpace(f.Title) == "" {
		errs = append(errs, "library.errors.titleRequired")
	}
	errs = append(errs, validateCore(f.MediaType, f.Bucket, f.Format)...)
	if len(errs) > 0 {
		return primitive.NilObjectID, errs, ErrValidation
	}

	title := strings.TrimSpace(f.Title)
	sortTitle := strings.ToLower(title)
	var year *int32
	if f.Year != nil {
		y := int32(*f.Year)
		year = &y
	}

	mediaRec := &domainlib.MediaRecord{
		Source:            domainlib.SourceManual,
		MediaType:         f.MediaType,
		Title:             title,
		SortTitle:         &sortTitle,
		Year:              year,
		BarcodeCandidates: []string{},
	}
	if err := s.media.Insert(ctx, mediaRec); err != nil {
		return primitive.NilObjectID, nil, err
	}

	var formatPtr *string
	if strings.TrimSpace(f.Format) != "" {
		fv := strings.TrimSpace(f.Format)
		formatPtr = &fv
	}

	libEntry := &domainlib.LibraryEntry{
		UserID:        userID,
		MediaRecordID: mediaRec.ID,
		Bucket:        f.Bucket,
		MediaType:     f.MediaType,
		Format:        formatPtr,
		Barcode:       optionalString(f.Barcode),
		PurchaseDate:  optionalString(f.PurchaseDate),
		Notes:         optionalString(f.Notes),
		Tags:          parseTags(f.Tags),
	}

	if err := s.entries.Insert(ctx, libEntry); err != nil {
		_ = s.media.DeleteByID(ctx, mediaRec.ID)
		if mongo.IsDuplicateKeyError(err) {
			return primitive.NilObjectID, []string{"library.errors.duplicateEntry"}, ErrValidation
		}
		return primitive.NilObjectID, nil, err
	}

	return libEntry.ID, nil, nil
}

// AttachFromMediaRecord creates a library entry for an existing media record (e.g. local barcode match), or
// returns the existing entry id for the same user, media, and bucket. Optional barcode is stored on a newly inserted entry.
// wasExisting is true when a row for that user, media, and bucket was already present.
func (s *Service) AttachFromMediaRecord(ctx context.Context, userID, mediaRecordID primitive.ObjectID, bucket, barcode string) (entryID primitive.ObjectID, wasExisting bool, formErrs []string, err error) {
	bucket = strings.TrimSpace(bucket)
	mediaRec, err := s.media.FindByID(ctx, mediaRecordID)
	if err != nil {
		return primitive.NilObjectID, false, nil, err
	}
	if mediaRec == nil {
		return primitive.NilObjectID, false, []string{"library.errors.mediaNotFound"}, ErrValidation
	}

	var errs []string
	errs = append(errs, validateCore(mediaRec.MediaType, bucket, "")...)
	if len(errs) > 0 {
		return primitive.NilObjectID, false, errs, ErrValidation
	}

	entry, err := s.entries.FindByUserMediaBucketAndFormat(ctx, userID, mediaRecordID, bucket, nil)
	if err != nil {
		return primitive.NilObjectID, false, nil, err
	}
	if entry != nil {
		return entry.ID, true, nil, nil
	}

	libEntry := &domainlib.LibraryEntry{
		UserID:        userID,
		MediaRecordID: mediaRecordID,
		Bucket:        bucket,
		MediaType:     mediaRec.MediaType,
		Barcode:       optionalString(barcode),
	}
	if err := s.entries.Insert(ctx, libEntry); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, lookupErr := s.entries.FindByUserMediaBucketAndFormat(ctx, userID, mediaRecordID, bucket, nil)
			if lookupErr != nil {
				return primitive.NilObjectID, false, nil, lookupErr
			}
			if existing != nil {
				return existing.ID, true, nil, nil
			}
			return primitive.NilObjectID, false, []string{"library.errors.duplicateEntry"}, ErrValidation
		}
		return primitive.NilObjectID, false, nil, err
	}
	return libEntry.ID, false, nil, nil
}

func (s *Service) UpdateEntry(ctx context.Context, userID, entryID primitive.ObjectID, f ManualUpdateForm) ([]string, error) {
	entry, err := s.entries.FindByIDAndUser(ctx, entryID, userID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, ErrNotFound
	}

	mediaRec, err := s.media.FindByID(ctx, entry.MediaRecordID)
	if err != nil {
		return nil, err
	}
	if mediaRec == nil {
		return nil, ErrNotFound
	}

	var errs []string
	if mediaRec.Source == domainlib.SourceManual {
		if strings.TrimSpace(f.Title) == "" {
			errs = append(errs, "library.errors.titleRequired")
		}
		errs = append(errs, validateCore(f.MediaType, f.Bucket, f.Format)...)
	} else {
		errs = append(errs, validateCore(mediaRec.MediaType, f.Bucket, f.Format)...)
	}
	if len(errs) > 0 {
		return errs, ErrValidation
	}

	if mediaRec.Source == domainlib.SourceManual {
		title := strings.TrimSpace(f.Title)
		sortTitle := strings.ToLower(title)
		var year *int32
		if f.Year != nil {
			y := int32(*f.Year)
			year = &y
		}
		if err := s.media.UpdateManualCore(ctx, mediaRec.ID, title, &sortTitle, f.MediaType, year); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		entry.MediaType = f.MediaType
	} else {
		entry.MediaType = mediaRec.MediaType
	}

	var formatPtr *string
	if strings.TrimSpace(f.Format) != "" {
		fv := strings.TrimSpace(f.Format)
		formatPtr = &fv
	}

	entry.Bucket = f.Bucket
	entry.Format = formatPtr
	entry.Barcode = optionalString(f.Barcode)
	entry.PurchaseDate = optionalString(f.PurchaseDate)
	entry.Notes = optionalString(f.Notes)
	entry.Tags = parseTags(f.Tags)

	if err := s.entries.Update(ctx, entry); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return []string{"library.errors.duplicateEntry"}, ErrValidation
		}
		return nil, fmt.Errorf("update library entry: %w", err)
	}

	return nil, nil
}

func (s *Service) DeleteEntry(ctx context.Context, userID, entryID primitive.ObjectID) error {
	entry, err := s.entries.FindByIDAndUser(ctx, entryID, userID)
	if err != nil {
		return err
	}
	if entry == nil {
		return ErrNotFound
	}

	mediaID := entry.MediaRecordID
	if err := s.entries.DeleteByIDAndUser(ctx, entryID, userID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	n, err := s.entries.CountByMediaRecordID(ctx, mediaID)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	mediaRec, err := s.media.FindByID(ctx, mediaID)
	if err != nil {
		return err
	}
	if mediaRec == nil {
		return nil
	}
	if mediaRec.Source != domainlib.SourceManual {
		return nil
	}
	_ = s.media.DeleteByID(ctx, mediaID)
	return nil
}
