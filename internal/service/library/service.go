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

func (s *Service) ListBucket(ctx context.Context, userID primitive.ObjectID, bucket string) ([]EntryWithMedia, error) {
	if !domainlib.IsBucket(bucket) {
		return nil, fmt.Errorf("bucket: %w", ErrValidation)
	}
	entries, err := s.entries.ListByUserAndBucket(ctx, userID, bucket)
	if err != nil {
		return nil, err
	}
	return s.attachMedia(ctx, entries)
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
