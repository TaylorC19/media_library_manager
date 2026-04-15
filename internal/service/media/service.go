package media

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"media_library_manager/internal/config"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/providers/musicbrainz"
	"media_library_manager/internal/providers/openlibrary"
	"media_library_manager/internal/providers/rawg"
	"media_library_manager/internal/providers/tmdb"
	"media_library_manager/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
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
	tmdb    *tmdb.Client
	mb      *musicbrainz.Client
	ol      *openlibrary.Client
	rawg    *rawg.Client
}

type ImportForm struct {
	Bucket     string
	MediaType  string
	Provider   string
	ExternalID string
	TMDBKind   string
}

type ImportOutcome struct {
	MediaRecord             *domainlib.MediaRecord
	LibraryEntry            *domainlib.LibraryEntry
	WasExistingMediaRecord  bool
	WasExistingLibraryEntry bool
}

type RefreshOutcome struct {
	MediaRecord           *domainlib.MediaRecord `json:"mediaRecord"`
	WasRefreshed          bool                   `json:"wasRefreshed"`
	RefreshedFromProvider string                 `json:"refreshedFromProvider,omitempty"`
	UnavailableReason     string                 `json:"unavailableReason,omitempty"`
}

func NewService(cfg config.Config, media *repository.MediaRecordsRepository, entries *repository.LibraryEntriesRepository) *Service {
	return &Service{
		media:   media,
		entries: entries,
		tmdb: &tmdb.Client{
			APIKey: cfg.TMDBAPIKey,
			HTTP:   tmdb.DefaultHTTPClient(),
		},
		mb: &musicbrainz.Client{
			UserAgent: cfg.MusicBrainzUA,
			HTTP:      musicbrainz.DefaultHTTPClient(),
		},
		ol: &openlibrary.Client{HTTP: openlibrary.DefaultHTTPClient()},
		rawg: &rawg.Client{
			APIKey: cfg.RAWGAPIKey,
			HTTP:   rawg.DefaultHTTPClient(),
		},
	}
}

func (s *Service) Import(ctx context.Context, userID primitive.ObjectID, form ImportForm) (*ImportOutcome, []string, error) {
	form.Provider = strings.TrimSpace(form.Provider)
	form.ExternalID = strings.TrimSpace(form.ExternalID)
	form.MediaType = strings.TrimSpace(form.MediaType)
	form.Bucket = strings.TrimSpace(form.Bucket)
	form.TMDBKind = strings.TrimSpace(form.TMDBKind)

	errs := validateImportForm(form)
	if len(errs) > 0 {
		return nil, errs, ErrValidation
	}

	existingByProviderRef, err := s.media.FindByProviderImportRef(ctx, form.Provider, form.ExternalID, form.TMDBKind)
	if err != nil {
		return nil, nil, err
	}
	if existingByProviderRef != nil {
		return s.attachLibraryEntry(ctx, userID, existingByProviderRef, form.Bucket, true)
	}

	normalized, err := s.fetchNormalizedRecord(ctx, form)
	if err != nil {
		return nil, nil, err
	}

	reusable, err := s.findReusableRecord(ctx, normalized)
	if err != nil {
		return nil, nil, err
	}

	if reusable == nil {
		if err := s.media.Insert(ctx, normalized); err != nil {
			return nil, nil, err
		}
		return s.attachLibraryEntry(ctx, userID, normalized, form.Bucket, false)
	}

	merged := mergeProviderData(reusable, normalized)
	if err := s.media.UpdateProviderData(ctx, merged); err != nil {
		return nil, nil, err
	}
	return s.attachLibraryEntry(ctx, userID, merged, form.Bucket, true)
}

func (s *Service) RefreshScaffold(ctx context.Context, mediaRecordID primitive.ObjectID) (*RefreshOutcome, error) {
	rec, err := s.media.FindByID(ctx, mediaRecordID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrNotFound
	}

	provider, providerID := refreshProviderRef(rec)
	if provider == "" || providerID == "" {
		return &RefreshOutcome{
			MediaRecord:       rec,
			WasRefreshed:      false,
			UnavailableReason: "provider_ref_unavailable",
		}, nil
	}

	return &RefreshOutcome{
		MediaRecord:           rec,
		WasRefreshed:          false,
		RefreshedFromProvider: provider,
		UnavailableReason:     "refresh_not_implemented",
	}, nil
}

func (s *Service) attachLibraryEntry(ctx context.Context, userID primitive.ObjectID, mediaRec *domainlib.MediaRecord, bucket string, wasExistingMediaRecord bool) (*ImportOutcome, []string, error) {
	outcome := &ImportOutcome{
		MediaRecord:            mediaRec,
		WasExistingMediaRecord: wasExistingMediaRecord,
	}

	if bucket == "" {
		return outcome, nil, nil
	}

	entry, err := s.entries.FindByUserMediaBucketAndFormat(ctx, userID, mediaRec.ID, bucket, nil)
	if err != nil {
		return nil, nil, err
	}
	if entry != nil {
		outcome.LibraryEntry = entry
		outcome.WasExistingLibraryEntry = true
		return outcome, nil, nil
	}

	entry = &domainlib.LibraryEntry{
		UserID:        userID,
		MediaRecordID: mediaRec.ID,
		Bucket:        bucket,
		MediaType:     mediaRec.MediaType,
	}
	if err := s.entries.Insert(ctx, entry); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, lookupErr := s.entries.FindByUserMediaBucketAndFormat(ctx, userID, mediaRec.ID, bucket, nil)
			if lookupErr != nil {
				return nil, nil, lookupErr
			}
			if existing != nil {
				outcome.LibraryEntry = existing
				outcome.WasExistingLibraryEntry = true
				return outcome, nil, nil
			}
			return nil, []string{"library.errors.duplicateEntry"}, ErrValidation
		}
		return nil, nil, err
	}

	outcome.LibraryEntry = entry
	return outcome, nil, nil
}

func (s *Service) fetchNormalizedRecord(ctx context.Context, form ImportForm) (*domainlib.MediaRecord, error) {
	switch form.Provider {
	case "tmdb":
		detail, err := s.tmdb.GetDetails(ctx, form.ExternalID, form.TMDBKind)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			return nil, ErrNotFound
		}
		return normalizeTMDBRecord(detail), nil
	case "musicbrainz":
		detail, err := s.mb.GetReleaseDetails(ctx, form.ExternalID)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			return nil, ErrNotFound
		}
		return normalizeMusicBrainzRecord(detail), nil
	case "open_library":
		detail, err := s.ol.GetWorkDetails(ctx, form.ExternalID)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			return nil, ErrNotFound
		}
		return normalizeOpenLibraryRecord(detail), nil
	case "rawg":
		detail, err := s.rawg.GetGameDetails(ctx, form.ExternalID)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			return nil, ErrNotFound
		}
		return normalizeRAWGRecord(detail), nil
	default:
		return nil, ErrValidation
	}
}

func (s *Service) findReusableRecord(ctx context.Context, normalized *domainlib.MediaRecord) (*domainlib.MediaRecord, error) {
	if normalized == nil {
		return nil, nil
	}

	if supportsBarcodeDedupe(normalized.MediaType) && len(normalized.BarcodeCandidates) > 0 {
		records, err := s.media.FindByAnyBarcodeCandidates(ctx, normalized.MediaType, normalized.BarcodeCandidates)
		if err != nil {
			return nil, err
		}
		if unique := singleUniqueRecord(records); unique != nil {
			return unique, nil
		}
	}

	creator := primaryCreator(normalized)
	if creator == "" || normalized.Year == nil {
		return nil, nil
	}

	candidates, err := s.media.FindLooseTitleYear(ctx, normalized.MediaType, normalized.Title, normalized.Year)
	if err != nil {
		return nil, err
	}

	var matches []domainlib.MediaRecord
	for _, candidate := range candidates {
		if normalizeTitle(candidate.Title) != normalizeTitle(normalized.Title) {
			continue
		}
		if candidate.Year == nil || *candidate.Year != *normalized.Year {
			continue
		}
		if primaryCreator(&candidate) != creator {
			continue
		}
		matches = append(matches, candidate)
	}

	return singleUniqueRecord(matches), nil
}

func validateImportForm(form ImportForm) []string {
	var errs []string
	if strings.TrimSpace(form.ExternalID) == "" {
		errs = append(errs, "search.import.externalIdRequired")
	}
	if !allowedProvider(form.Provider) {
		errs = append(errs, "search.import.invalidProvider")
	}
	if !domainlib.IsMediaType(form.MediaType) {
		errs = append(errs, "library.errors.invalidMediaType")
	}
	if form.Bucket != "" && !domainlib.IsBucket(form.Bucket) {
		errs = append(errs, "library.errors.invalidBucket")
	}

	switch form.Provider {
	case "tmdb":
		if form.TMDBKind != "movie" && form.TMDBKind != "tv" {
			errs = append(errs, "search.import.invalidTMDBKind")
		}
		if form.MediaType != form.TMDBKind {
			errs = append(errs, "search.import.mediaTypeMismatch")
		}
	case "musicbrainz":
		if form.MediaType != "album" {
			errs = append(errs, "search.import.mediaTypeMismatch")
		}
	case "open_library":
		if form.MediaType != "book" {
			errs = append(errs, "search.import.mediaTypeMismatch")
		}
	case "rawg":
		if form.MediaType != "game" {
			errs = append(errs, "search.import.mediaTypeMismatch")
		}
	}

	return errs
}

func allowedProvider(provider string) bool {
	switch provider {
	case "tmdb", "musicbrainz", "open_library", "rawg":
		return true
	default:
		return false
	}
}

func normalizeTMDBRecord(detail *tmdb.Detail) *domainlib.MediaRecord {
	title := strings.TrimSpace(detail.Title)
	sortTitle := normalizeTitle(title)
	releaseDate := trimmedPtr(detail.ReleaseDate)
	summary := trimmedPtr(detail.Overview)
	now := time.Now().UTC()

	rec := &domainlib.MediaRecord{
		Source:            domainlib.SourceProvider,
		MediaType:         detail.Kind,
		Title:             title,
		SortTitle:         &sortTitle,
		ReleaseDate:       releaseDate,
		Year:              int32PtrFromInt(tmdb.YearFromDate(detail.ReleaseDate)),
		ImageURL:          tmdb.PosterURL(detail.PosterPath),
		Summary:           summary,
		ProviderRefs:      bson.M{"tmdb": bson.M{"id": detail.ExternalID(), "kind": detail.Kind}},
		ExternalRatings:   numericMap("tmdb", detail.VoteAverage),
		BarcodeCandidates: []string{},
		Details:           bson.M{},
		LastSyncedAt:      &now,
	}

	switch detail.Kind {
	case "movie":
		rec.Details = bson.M{
			"directors":      normalizeStrings(detail.Directors),
			"cast":           normalizeStrings(detail.Cast),
			"genres":         normalizeStrings(detail.Genres),
			"runtimeMinutes": int32PtrFromInt(detail.RuntimeMinutes),
		}
	case "tv":
		rec.Details = bson.M{
			"creators": normalizeStrings(detail.Creators),
			"genres":   normalizeStrings(detail.Genres),
			"seasons":  int32PtrFromInt(detail.Seasons),
			"episodes": int32PtrFromInt(detail.Episodes),
		}
	}

	return rec
}

func normalizeMusicBrainzRecord(detail *musicbrainz.ReleaseDetails) *domainlib.MediaRecord {
	title := strings.TrimSpace(detail.Title)
	sortTitle := normalizeTitle(title)
	now := time.Now().UTC()
	barcodeCandidates := normalizeBarcodes([]string{detail.Barcode})

	return &domainlib.MediaRecord{
		Source:            domainlib.SourceProvider,
		MediaType:         "album",
		Title:             title,
		SortTitle:         &sortTitle,
		ReleaseDate:       trimmedPtr(detail.Date),
		Year:              int32PtrFromInt(musicbrainz.YearFromPartialDate(detail.Date)),
		Summary:           nil,
		ProviderRefs:      bson.M{"musicbrainz": bson.M{"releaseId": detail.ID}},
		BarcodeCandidates: barcodeCandidates,
		Details: bson.M{
			"artists":        normalizeStrings(detail.Artists),
			"label":          trimmedPtr(detail.Label),
			"catalogNumber":  trimmedPtr(detail.CatalogNumber),
			"releaseCountry": trimmedPtr(detail.Country),
			"trackCount":     int32PtrFromInt(detail.TrackCount),
		},
		LastSyncedAt: &now,
	}
}

func normalizeOpenLibraryRecord(detail *openlibrary.WorkDetails) *domainlib.MediaRecord {
	title := strings.TrimSpace(detail.Title)
	sortTitle := normalizeTitle(title)
	now := time.Now().UTC()
	summary := trimmedPtr(detail.Description)
	releaseDate := trimmedPtr(detail.FirstPublishDate)

	return &domainlib.MediaRecord{
		Source:            domainlib.SourceProvider,
		MediaType:         "book",
		Title:             title,
		SortTitle:         &sortTitle,
		ReleaseDate:       releaseDate,
		Year:              int32PtrFromInt(yearFromDatePrefix(detail.FirstPublishDate)),
		ImageURL:          openlibrary.CoverURLFromID(detail.CoverID),
		Summary:           summary,
		ProviderRefs:      bson.M{"openLibrary": bson.M{"workKey": detail.WorkKey}},
		BarcodeCandidates: []string{},
		Details: bson.M{
			"authors": normalizeStrings(detail.Authors),
		},
		LastSyncedAt: &now,
	}
}

func normalizeRAWGRecord(detail *rawg.GameDetails) *domainlib.MediaRecord {
	title := strings.TrimSpace(detail.Name)
	sortTitle := normalizeTitle(title)
	now := time.Now().UTC()

	return &domainlib.MediaRecord{
		Source:            domainlib.SourceProvider,
		MediaType:         "game",
		Title:             title,
		SortTitle:         &sortTitle,
		ReleaseDate:       trimmedPtr(detail.Released),
		Year:              int32PtrFromInt(yearFromDatePrefix(detail.Released)),
		ImageURL:          rawg.BackgroundURL(detail.BackgroundImage),
		Summary:           trimmedPtr(detail.Description),
		ProviderRefs:      bson.M{"rawg": bson.M{"id": strconv.Itoa(detail.ID)}},
		ExternalRatings:   numericMap("rawg", detail.Rating),
		BarcodeCandidates: []string{},
		Details: bson.M{
			"developers": normalizeStrings(detail.Developers),
			"publishers": normalizeStrings(detail.Publishers),
			"platforms":  normalizeStrings(detail.Platforms),
			"genres":     normalizeStrings(detail.Genres),
		},
		LastSyncedAt: &now,
	}
}

func mergeProviderData(existing, incoming *domainlib.MediaRecord) *domainlib.MediaRecord {
	merged := *existing
	merged.Source = domainlib.SourceProvider
	merged.MediaType = incoming.MediaType
	merged.Title = incoming.Title
	merged.SortTitle = coalesceStringPtr(incoming.SortTitle, existing.SortTitle)
	merged.ReleaseDate = coalesceStringPtr(incoming.ReleaseDate, existing.ReleaseDate)
	merged.Year = coalesceInt32Ptr(incoming.Year, existing.Year)
	merged.ImageURL = coalesceStringPtr(incoming.ImageURL, existing.ImageURL)
	merged.Summary = coalesceStringPtr(incoming.Summary, existing.Summary)
	merged.ProviderRefs = mergeMaps(existing.ProviderRefs, incoming.ProviderRefs)
	merged.ExternalRatings = mergeMaps(existing.ExternalRatings, incoming.ExternalRatings)
	merged.BarcodeCandidates = mergeStringLists(existing.BarcodeCandidates, incoming.BarcodeCandidates)
	merged.Details = mergeDetails(existing.MediaType, existing.Details, incoming.Details)
	merged.LastSyncedAt = incoming.LastSyncedAt
	return &merged
}

func mergeDetails(mediaType string, existing, incoming bson.M) bson.M {
	if existing == nil && incoming == nil {
		return bson.M{}
	}
	out := bson.M{}
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range incoming {
		switch value := v.(type) {
		case []string:
			out[k] = mergeStringLists(asStringSlice(out[k]), value)
		default:
			if value != nil {
				out[k] = value
			}
		}
	}
	if mediaType == "book" {
		out["authors"] = mergeStringLists(asStringSlice(existing["authors"]), asStringSlice(incoming["authors"]))
	}
	return out
}

func primaryCreator(rec *domainlib.MediaRecord) string {
	if rec == nil {
		return ""
	}
	switch rec.MediaType {
	case "movie":
		return normalizeTitle(firstString(rec.Details, "directors"))
	case "tv":
		return normalizeTitle(firstString(rec.Details, "creators"))
	case "album":
		return normalizeTitle(firstString(rec.Details, "artists"))
	case "book":
		return normalizeTitle(firstString(rec.Details, "authors"))
	case "game":
		return normalizeTitle(firstString(rec.Details, "developers"))
	default:
		return ""
	}
}

func firstString(details bson.M, key string) string {
	values := asStringSlice(details[key])
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func refreshProviderRef(rec *domainlib.MediaRecord) (string, string) {
	if rec == nil || rec.ProviderRefs == nil {
		return "", ""
	}
	if tmdbRef, ok := rec.ProviderRefs["tmdb"].(bson.M); ok {
		id := strings.TrimSpace(stringMapValue(tmdbRef, "id"))
		if id != "" {
			return "tmdb", id
		}
	}
	if mbRef, ok := rec.ProviderRefs["musicbrainz"].(bson.M); ok {
		id := strings.TrimSpace(stringMapValue(mbRef, "releaseId"))
		if id != "" {
			return "musicbrainz", id
		}
	}
	if olRef, ok := rec.ProviderRefs["openLibrary"].(bson.M); ok {
		id := strings.TrimSpace(stringMapValue(olRef, "workKey"))
		if id != "" {
			return "open_library", id
		}
	}
	if rawgRef, ok := rec.ProviderRefs["rawg"].(bson.M); ok {
		id := strings.TrimSpace(stringMapValue(rawgRef, "id"))
		if id != "" {
			return "rawg", id
		}
	}
	return "", ""
}

func supportsBarcodeDedupe(mediaType string) bool {
	return mediaType == "album" || mediaType == "book"
}

func singleUniqueRecord(records []domainlib.MediaRecord) *domainlib.MediaRecord {
	if len(records) == 0 {
		return nil
	}
	seen := make(map[primitive.ObjectID]domainlib.MediaRecord)
	for _, record := range records {
		seen[record.ID] = record
	}
	if len(seen) != 1 {
		return nil
	}
	for _, record := range seen {
		rec := record
		return &rec
	}
	return nil
}

func normalizeTitle(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func normalizeStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizeBarcodes(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func mergeStringLists(existing, incoming []string) []string {
	return normalizeStrings(append(existing, incoming...))
}

func mergeMaps(existing, incoming bson.M) bson.M {
	if existing == nil && incoming == nil {
		return nil
	}
	out := bson.M{}
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range incoming {
		out[k] = v
	}
	return out
}

func asStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string{}, v...)
	case primitive.A:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func numericMap(key string, value *float64) bson.M {
	if value == nil {
		return nil
	}
	return bson.M{key: *value}
}

func int32PtrFromInt(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}

func trimmedPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func yearFromDatePrefix(value string) *int {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return nil
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return nil
	}
	return &year
}

func coalesceStringPtr(incoming, existing *string) *string {
	if incoming != nil && strings.TrimSpace(*incoming) != "" {
		return incoming
	}
	return existing
}

func coalesceInt32Ptr(incoming, existing *int32) *int32 {
	if incoming != nil {
		return incoming
	}
	return existing
}

func stringMapValue(m bson.M, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}
