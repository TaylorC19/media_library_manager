package barcode

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"media_library_manager/internal/config"
	bdom "media_library_manager/internal/domain/barcode"
	domainlib "media_library_manager/internal/domain/library"
	"media_library_manager/internal/providers/discogs"
	"media_library_manager/internal/providers/musicbrainz"
	"media_library_manager/internal/providers/openlibrary"
	"media_library_manager/internal/providers/providererrors"
	"media_library_manager/internal/repository"
	"media_library_manager/internal/service/reliability"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ErrInvalidBarcode is returned when input normalizes to an empty string.
var ErrInvalidBarcode = errors.New("barcode must contain digits or X per ISBN rules after normalization")

const providerResultLimit = 10

// Service orchestrates local + provider barcode lookup (lookup only; no writes to media/ library).
type Service struct {
	cfg         config.Config
	media       *repository.MediaRecordsRepository
	lib         *repository.LibraryEntriesRepository
	scan        *repository.ScanLogsRepository
	cache       *reliability.Cache
	reliability *reliability.Service
	ol          *openlibrary.Client
	mb          *musicbrainz.Client
	disc        *discogs.Client
}

func NewService(cfg config.Config, media *repository.MediaRecordsRepository, lib *repository.LibraryEntriesRepository, scan *repository.ScanLogsRepository, cache *reliability.Cache, reliabilitySvc *reliability.Service, mbThrottle *musicbrainz.Throttle) *Service {
	return &Service{
		cfg:         cfg,
		media:       media,
		lib:         lib,
		scan:        scan,
		cache:       cache,
		reliability: reliabilitySvc,
		ol:          &openlibrary.Client{HTTP: openlibrary.DefaultHTTPClient()},
		mb: &musicbrainz.Client{
			UserAgent: cfg.MusicBrainzUA,
			HTTP:      musicbrainz.DefaultHTTPClient(),
			Throttle:  mbThrottle,
		},
		disc: &discogs.Client{
			Token: cfg.DiscogsToken,
			HTTP:  discogs.DefaultHTTPClient(),
		},
	}
}

// Lookup runs normalized lookup: local first, then ordered provider stages.
func (s *Service) Lookup(ctx context.Context, userID primitive.ObjectID, req bdom.Request) (*bdom.Response, error) {
	norm := NormalizeBarcode(req.Barcode)
	if norm == "" {
		return nil, ErrInvalidBarcode
	}

	pref := strings.TrimSpace(req.PreferredMediaType)
	if pref != "" && !domainlib.IsMediaType(pref) {
		pref = ""
	}

	hint := InferHint(norm, pref)

	var localFilter *string
	if hint.PreferredMediaType != nil {
		localFilter = hint.PreferredMediaType
	}

	locals, err := s.buildLocalCandidates(ctx, userID, norm, localFilter)
	if err != nil {
		return nil, err
	}

	failures := make([]bdom.Failure, 0, 4)
	seenFail := make(map[string]struct{})

	var providers []scoredCandidate
	stages := BuildStages(hint)
	stageNumber := 0
	for _, st := range stages {
		stageNumber++
		for _, prov := range st.Providers {
			switch prov {
			case bdom.ProviderOpenLibrary:
				if st.MediaType != "book" {
					continue
				}
				cands, fails := s.openLibraryByISBN(ctx, norm, stageNumber, st.RankTier)
				for _, f := range fails {
					addFailure(&failures, seenFail, f)
				}
				providers = append(providers, cands...)

			case bdom.ProviderDiscogs:
				if st.MediaType != "album" {
					continue
				}
				if strings.TrimSpace(s.cfg.DiscogsToken) == "" {
					addFailure(&failures, seenFail, bdom.Failure{Provider: bdom.ProviderDiscogs, Code: bdom.CodeUnsupported})
					continue
				}
				cands, fail := s.discogsByBarcode(ctx, norm, stageNumber, st.RankTier)
				if fail != nil {
					addFailure(&failures, seenFail, *fail)
					continue
				}
				providers = append(providers, cands...)

			case bdom.ProviderMusicBrainz:
				if st.MediaType != "album" {
					continue
				}
				cands, fail := s.musicBrainzByBarcode(ctx, norm, stageNumber, st.RankTier)
				if fail != nil {
					addFailure(&failures, seenFail, *fail)
					continue
				}
				providers = append(providers, cands...)
			}
		}
	}

	ranked := rankAll(locals, providers)
	out := make([]bdom.Candidate, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.c)
	}

	mediaT := resolveMediaType(out, hint)
	fb := buildFallback(out, failures, hint, mediaT, norm)

	resp := &bdom.Response{
		Barcode:    norm,
		MediaType:  mediaT,
		Candidates: out,
		Failures:   failures,
		Fallback:   fb,
	}

	s.logScan(ctx, userID, norm, out)

	return resp, nil
}

type scoredCandidate struct {
	c         bdom.Candidate
	sortTier  int
	stageIdx  int
	hasLinked bool
	sortKey   string
}

func (s *Service) buildLocalCandidates(ctx context.Context, userID primitive.ObjectID, norm string, mediaType *string) ([]scoredCandidate, error) {
	recs, err := s.media.FindByBarcodeCandidate(ctx, norm, mediaType)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, nil
	}
	ids := make([]primitive.ObjectID, 0, len(recs))
	for _, r := range recs {
		ids = append(ids, r.ID)
	}
	entries, err := s.lib.FindByUserAndMediaRecordIDs(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byMedia := make(map[primitive.ObjectID][]domainlib.LibraryEntry)
	for _, e := range entries {
		byMedia[e.MediaRecordID] = append(byMedia[e.MediaRecordID], e)
	}

	var out []scoredCandidate
	for _, rec := range recs {
		entList := byMedia[rec.ID]
		linked := make([]bdom.LinkedEntry, 0, len(entList))
		for _, e := range entList {
			linked = append(linked, bdom.LinkedEntry{EntryID: e.ID.Hex(), Bucket: e.Bucket})
		}
		cl := creatorLine(&rec)
		c := bdom.Candidate{
			Source:                "local",
			MediaRecordID:         rec.ID.Hex(),
			MediaType:             rec.MediaType,
			Title:                 rec.Title,
			Year:                  rec.Year,
			ImageURL:              rec.ImageURL,
			CreatorLine:           cl,
			HasLinkedLibraryEntry: len(linked) > 0,
			LinkedLibraryEntries:  linked,
		}
		out = append(out, scoredCandidate{
			c:         c,
			sortTier:  0,
			stageIdx:  0,
			hasLinked: len(linked) > 0,
			sortKey:   "local:" + rec.ID.Hex(),
		})
	}
	return out, nil
}

func (s *Service) openLibraryByISBN(ctx context.Context, norm string, stageIdx, rankTier int) ([]scoredCandidate, []bdom.Failure) {
	hits, err := reliability.Wrap(ctx, s.cache, reliability.OperationBarcode, bdom.ProviderOpenLibrary, s.reliability.BarcodeCacheKey(strPtr("book"), norm), func(ctx context.Context) ([]openlibrary.WorkHit, error) {
		return s.ol.SearchByISBN(ctx, norm, providerResultLimit)
	})
	if err != nil {
		f := mapProviderErr(bdom.ProviderOpenLibrary, err)
		return nil, []bdom.Failure{f}
	}
	var out []scoredCandidate
	seen := make(map[string]struct{})
	for _, h := range hits {
		if len(out) >= providerResultLimit {
			break
		}
		key := strings.TrimPrefix(strings.TrimSpace(h.Key), "/works/")
		key = strings.TrimPrefix(key, "works/")
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		pair := bdom.ProviderOpenLibrary + ":" + key
		if _, ok := seen[pair]; ok {
			continue
		}
		seen[pair] = struct{}{}
		year := int32PtrFromInt(h.FirstYear)
		img := openlibrary.CoverURLFromID(h.CoverID)
		cl := strPtr(openlibrary.JoinAuthors(h.AuthorNames))
		c := bdom.Candidate{
			Source:      "provider",
			Provider:    bdom.ProviderOpenLibrary,
			ProviderID:  key,
			MediaType:   "book",
			Title:       h.Title,
			Year:        year,
			ImageURL:    img,
			CreatorLine: cl,
		}
		tier := rankTier
		out = append(out, scoredCandidate{c: c, sortTier: tier, stageIdx: stageIdx, hasLinked: false, sortKey: "provider:openlibrary:" + key})
	}
	return out, nil
}

func (s *Service) discogsByBarcode(ctx context.Context, norm string, stageIdx, rankTier int) ([]scoredCandidate, *bdom.Failure) {
	hits, err := reliability.Wrap(ctx, s.cache, reliability.OperationBarcode, bdom.ProviderDiscogs, s.reliability.BarcodeCacheKey(strPtr("album"), norm), func(ctx context.Context) ([]discogs.SearchHit, error) {
		return s.disc.SearchByBarcode(ctx, norm, providerResultLimit)
	})
	if err != nil {
		f := mapProviderErr(bdom.ProviderDiscogs, err)
		return nil, &f
	}
	var out []scoredCandidate
	seen := make(map[string]struct{})
	for _, h := range hits {
		pair := bdom.ProviderDiscogs + ":" + strconv.Itoa(h.ID)
		if _, ok := seen[pair]; ok {
			continue
		}
		seen[pair] = struct{}{}
		_, title := splitDiscogsTitle(h.Title)
		if title == "" {
			title = h.Title
		}
		artists, _ := splitDiscogsTitle(h.Title)
		cl := strPtr(artists)
		if cl != nil && *cl == "" {
			cl = nil
		}
		tier := rankTier
		if hasExactBarcodes(norm, h.Barcode) {
			tier = 1
		}
		c := bdom.Candidate{
			Source:      "provider",
			Provider:    bdom.ProviderDiscogs,
			ProviderID:  strconv.Itoa(h.ID),
			MediaType:   "album",
			Title:       title,
			Year:        int32PtrFromInt(h.Year),
			ImageURL:    h.CoverImage,
			CreatorLine: cl,
		}
		out = append(out, scoredCandidate{c: c, sortTier: tier, stageIdx: stageIdx, hasLinked: false, sortKey: "provider:discogs:" + strconv.Itoa(h.ID)})
	}
	return out, nil
}

func (s *Service) musicBrainzByBarcode(ctx context.Context, norm string, stageIdx, rankTier int) ([]scoredCandidate, *bdom.Failure) {
	hits, err := reliability.Wrap(ctx, s.cache, reliability.OperationBarcode, bdom.ProviderMusicBrainz, s.reliability.BarcodeCacheKey(strPtr("album"), norm), func(ctx context.Context) ([]musicbrainz.ReleaseHit, error) {
		return s.mb.SearchByBarcode(ctx, norm)
	})
	if err != nil {
		f := mapProviderErr(bdom.ProviderMusicBrainz, err)
		return nil, &f
	}
	if len(hits) > providerResultLimit {
		hits = hits[:providerResultLimit]
	}
	var out []scoredCandidate
	seen := make(map[string]struct{})
	for _, h := range hits {
		pair := bdom.ProviderMusicBrainz + ":" + h.ID
		if _, ok := seen[pair]; ok {
			continue
		}
		seen[pair] = struct{}{}
		tier := rankTier
		if h.Barcode != "" && hasExactBarcodes(norm, []string{h.Barcode}) {
			tier = 1
		}
		y := musicbrainz.YearFromPartialDate(h.Date)
		var y32 *int32
		if y != nil {
			v := int32(*y)
			y32 = &v
		}
		cl := strPtr(h.ArtistName)
		if cl != nil && *cl == "" {
			cl = nil
		}
		c := bdom.Candidate{
			Source:      "provider",
			Provider:    bdom.ProviderMusicBrainz,
			ProviderID:  h.ID,
			MediaType:   "album",
			Title:       h.Title,
			Year:        y32,
			ImageURL:    nil,
			CreatorLine: cl,
		}
		out = append(out, scoredCandidate{c: c, sortTier: tier, stageIdx: stageIdx, hasLinked: false, sortKey: "provider:musicbrainz:" + h.ID})
	}
	return out, nil
}

func hasExactBarcodes(normalized string, cands []string) bool {
	for _, c := range cands {
		if NormalizeBarcode(c) == normalized {
			return true
		}
	}
	return false
}

func addFailure(out *[]bdom.Failure, seen map[string]struct{}, f bdom.Failure) {
	key := f.Provider + ":" + f.Code
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*out = append(*out, f)
}

func mapProviderErr(provider string, err error) bdom.Failure {
	if err == nil {
		return bdom.Failure{Provider: provider, Code: bdom.CodeUnavailable}
	}

	var perr *providererrors.Error
	if errors.As(err, &perr) {
		switch perr.Code {
		case providererrors.CodeTimeout:
			return bdom.Failure{Provider: provider, Code: bdom.CodeTimeout}
		case providererrors.CodeInvalidResponse:
			return bdom.Failure{Provider: provider, Code: bdom.CodeInvalidResponse}
		case providererrors.CodeUnsupported:
			return bdom.Failure{Provider: provider, Code: bdom.CodeUnsupported}
		default:
			return bdom.Failure{Provider: provider, Code: bdom.CodeUnavailable}
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return bdom.Failure{Provider: provider, Code: bdom.CodeTimeout}
	}
	return bdom.Failure{Provider: provider, Code: bdom.CodeUnavailable}
}

func rankAll(locals, providers []scoredCandidate) []scoredCandidate {
	all := append(append([]scoredCandidate{}, locals...), providers...)
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.sortTier != b.sortTier {
			return a.sortTier < b.sortTier
		}
		if a.hasLinked != b.hasLinked {
			return a.hasLinked
		}
		if a.stageIdx != b.stageIdx {
			return a.stageIdx < b.stageIdx
		}
		if a.c.Title != b.c.Title {
			return strings.Compare(a.c.Title, b.c.Title) < 0
		}
		ya, yb := yearKey(a.c.Year), yearKey(b.c.Year)
		if ya != yb {
			return ya > yb
		}
		return a.sortKey < b.sortKey
	})
	return all
}

func yearKey(y *int32) int {
	if y == nil {
		return -1
	}
	return int(*y)
}

func resolveMediaType(cands []bdom.Candidate, hint Hint) *string {
	if len(cands) > 0 {
		t := cands[0].MediaType
		return &t
	}
	if hint.PreferredMediaType != nil {
		return hint.PreferredMediaType
	}
	if hint.InferredMediaType != nil {
		return hint.InferredMediaType
	}
	return nil
}

func buildFallback(cands []bdom.Candidate, failures []bdom.Failure, hint Hint, mediaType *string, barcode string) *bdom.Fallback {
	if len(cands) > 0 {
		t := strings.TrimSpace(cands[0].Title)
		if t == "" {
			return &bdom.Fallback{Reason: bdom.ReasonManualConfirmation, ManualQuery: nil, MediaType: mediaType}
		}
		return &bdom.Fallback{Reason: bdom.ReasonManualConfirmation, ManualQuery: &t, MediaType: mediaType}
	}
	weakType := firstNonNilStr(hint.PreferredMediaType, hint.InferredMediaType)
	if weakType != nil && HasWeakCoverage(*weakType) {
		return &bdom.Fallback{Reason: bdom.ReasonWeakBarcodeCoverage, ManualQuery: nil, MediaType: mediaType}
	}
	if len(failures) > 0 {
		var mq *string
		if hint.IsIsbnLike {
			mq = &barcode
		}
		return &bdom.Fallback{Reason: bdom.ReasonProviderUnavailable, ManualQuery: mq, MediaType: mediaType}
	}
	var mq *string
	if hint.IsIsbnLike {
		mq = &barcode
	}
	return &bdom.Fallback{Reason: bdom.ReasonNoCandidates, ManualQuery: mq, MediaType: mediaType}
}

func firstNonNilStr(a, b *string) *string {
	if a != nil {
		return a
	}
	return b
}

func (s *Service) logScan(ctx context.Context, userID primitive.ObjectID, barcode string, cands []bdom.Candidate) {
	if s == nil || s.scan == nil {
		return
	}
	var mt *string
	var prov *string
	if len(cands) > 0 {
		t := cands[0].MediaType
		mt = &t
		if cands[0].Source == "provider" && cands[0].Provider != "" {
			p := cands[0].Provider
			prov = &p
		}
	}
	_ = s.scan.Insert(ctx, userID, barcode, mt, prov)
}

func splitDiscogsTitle(v string) (artist, title string) {
	v = strings.TrimSpace(v)
	parts := strings.SplitN(v, " - ", 2)
	if len(parts) < 2 {
		return "", v
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func creatorLine(rec *domainlib.MediaRecord) *string {
	if rec == nil || rec.Details == nil {
		return nil
	}
	var key string
	switch rec.MediaType {
	case "album":
		key = "artists"
	case "book":
		key = "authors"
	case "movie":
		key = "directors"
	case "tv":
		key = "creators"
	case "game":
		key = "developers"
	default:
		return nil
	}
	parts := stringSliceFromDetails(rec.Details, key)
	if len(parts) == 0 {
		return nil
	}
	s := strings.Join(parts, ", ")
	return &s
}

func stringSliceFromDetails(details bson.M, key string) []string {
	v, ok := details[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return append([]string{}, t...)
	case primitive.A:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []any:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func int32PtrFromInt(v *int) *int32 {
	if v == nil {
		return nil
	}
	x := int32(*v)
	return &x
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
