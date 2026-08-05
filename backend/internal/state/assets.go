// This file belongs to the session state layer.
// It owns domain asset state behavior for this package.
// It stores session truth; routing and interpretation decisions stay outside state structs.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// AssetKindProfileRevision identifies immutable birth-data revisions.
	AssetKindProfileRevision = "profile_revision"
	// AssetKindBaziChart identifies a BaZi natal-chart result.
	AssetKindBaziChart = "bazi_chart"
	// AssetKindZiweiChart identifies a ZiWei natal-chart result.
	AssetKindZiweiChart = "ziwei_chart"
	// AssetKindQimenChart identifies the legacy Qi Men event-chart alias.
	// New writes are normalized to AssetKindQimenCaseChart under an explicit Case.
	AssetKindQimenChart = "qimen_chart"
	// AssetKindQimenCaseChart identifies a QiMen chart bound to one exact Case.
	AssetKindQimenCaseChart = "qimen_case_chart"
	// AssetKindInterpretation identifies a follow-up-ready reading bound to one chart asset.
	AssetKindInterpretation = "interpretation"
)

// AssetRef identifies one exact persisted asset version. A reference, rather
// than a display label, is the only value execution code may use to reuse data.
type AssetRef struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// Subject is a stable consultation target within one session. Display is a
// conversational label such as "自己" or "孩子" and is not used as an asset key.
type Subject struct {
	ID        string    `json:"id"`
	Display   string    `json:"display"`
	CreatedAt time.Time `json:"created_at"`
}

// ProfileRevision preserves one normalized birth-data version for a Subject.
// A correction creates another revision instead of overwriting the old input.
type ProfileRevision struct {
	ID          string         `json:"id"`
	SubjectID   string         `json:"subject_id"`
	Version     int            `json:"version"`
	BirthData   map[string]any `json:"birth_data"`
	Fingerprint string         `json:"fingerprint"`
	Supersedes  string         `json:"supersedes,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Case represents one consultation or event scope. EventTime distinguishes
// event charts that belong to the same person but were cast at different times.
type Case struct {
	ID         string     `json:"id"`
	Domain     string     `json:"domain,omitempty"`
	Question   string     `json:"question,omitempty"`
	SubjectIDs []string   `json:"subject_ids,omitempty"`
	EventTime  *time.Time `json:"event_time,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// DomainAsset stores a typed domain result with complete input lineage. Payload
// remains a map during the migration because existing tools expose JSON maps.
type DomainAsset struct {
	Ref           AssetRef       `json:"ref"`
	OwnerKind     string         `json:"owner_kind"`
	OwnerID       string         `json:"owner_id"`
	SubjectIDs    []string       `json:"subject_ids,omitempty"`
	InputRefs     []AssetRef     `json:"input_refs,omitempty"`
	MethodVersion string         `json:"method_version,omitempty"`
	CalendarRule  string         `json:"calendar_rule,omitempty"`
	EffectiveTime *time.Time     `json:"effective_time,omitempty"`
	PayloadHash   string         `json:"payload_hash"`
	Payload       map[string]any `json:"payload"`
	CreatedAt     time.Time      `json:"created_at"`
}

// ActiveFocus records the only subject, profile and case allowed to supply
// implicit context for the current turn. It is a selection pointer, not a cache.
type ActiveFocus struct {
	SubjectIDs        []string   `json:"subject_ids,omitempty"`
	ProfileRevisionID string     `json:"profile_revision_id,omitempty"`
	CaseID            string     `json:"case_id,omitempty"`
	PrimaryAssetRefs  []AssetRef `json:"primary_asset_refs,omitempty"`
	ExcludedAssetRefs []AssetRef `json:"excluded_asset_refs,omitempty"`
}

// NormalizeSubjectLabel converts empty and whitespace-only labels into the
// session default label while preserving human-readable labels otherwise.
func NormalizeSubjectLabel(label string) string {
	if trimmed := strings.TrimSpace(label); trimmed != "" {
		return trimmed
	}
	return "自己"
}

// EnsureSubject returns the existing exact-label Subject or creates one. This
// intentionally does not apply semantic matching: mistaken identity is worse
// than a clarification in a deterministic domain workflow.
func (s *SessionState) EnsureSubject(label string) Subject {
	label = NormalizeSubjectLabel(label)
	for _, subject := range s.Subjects {
		if subject.Display == label {
			return subject
		}
	}
	subject := Subject{
		ID:        fmt.Sprintf("subject-%d", len(s.Subjects)+1),
		Display:   label,
		CreatedAt: time.Now().UTC(),
	}
	s.Subjects = append(s.Subjects, subject)
	return subject
}

// FindSubject returns an exact conversational-label match. Callers must ask for
// clarification instead of guessing when it returns false.
func (s *SessionState) FindSubject(label string) (Subject, bool) {
	label = strings.TrimSpace(label)
	for _, subject := range s.Subjects {
		if subject.Display == label {
			return subject, true
		}
	}
	return Subject{}, false
}

// SetActiveSubject switches the current selection without deleting any prior
// profile, chart, case or interpretation assets.
func (s *SessionState) SetActiveSubject(label string) Subject {
	s.MigrateLegacyAssets()
	subject := s.EnsureSubject(label)
	s.ActiveFocus.SubjectIDs = []string{subject.ID}
	s.Subject = subject.Display // legacy display projection for un-migrated callers
	if profile, ok := s.latestProfileForSubject(subject.ID); ok {
		s.ActiveFocus.ProfileRevisionID = profile.ID
	} else {
		s.ActiveFocus.ProfileRevisionID = ""
	}
	s.ActiveFocus.CaseID = ""
	s.ActiveFocus.PrimaryAssetRefs = nil
	s.ActiveFocus.ExcludedAssetRefs = nil
	s.syncLegacyActiveProjection()
	return subject
}

// ActiveSubject returns the selected Subject. Legacy sessions are migrated to a
// default "自己" subject before selection so callers always receive an identity.
func (s *SessionState) ActiveSubject() Subject {
	s.MigrateLegacyAssets()
	if len(s.ActiveFocus.SubjectIDs) > 0 {
		for _, subject := range s.Subjects {
			if subject.ID == s.ActiveFocus.SubjectIDs[0] {
				return subject
			}
		}
	}
	return s.SetActiveSubject("自己")
}

// MergeProfile creates a new ProfileRevision when a supplied field changes.
// Existing natal charts remain addressable through their older input revision.
func (s *SessionState) MergeProfile(patch map[string]any) bool {
	if s == nil || len(patch) == 0 {
		return false
	}
	s.MigrateLegacyAssets()
	subject := s.ActiveSubject()
	current, hasCurrent := s.activeProfile()
	next := map[string]any{}
	if hasCurrent {
		next = cloneMap(current.BirthData)
	}
	changed := false
	for key, value := range patch {
		if old, ok := next[key]; !ok || fmt.Sprint(old) != fmt.Sprint(value) {
			next[key] = value
			changed = true
		}
	}
	if !changed && hasCurrent {
		s.syncLegacyActiveProjection()
		return false
	}

	version := 1
	supersedes := ""
	if hasCurrent {
		version = current.Version + 1
		supersedes = current.ID
	}
	revision := ProfileRevision{
		ID:          fmt.Sprintf("profile-%s-v%d", subject.ID, version),
		SubjectID:   subject.ID,
		Version:     version,
		BirthData:   next,
		Fingerprint: profileFingerprint(next),
		Supersedes:  supersedes,
		CreatedAt:   time.Now().UTC(),
	}
	s.ProfileRevisions = append(s.ProfileRevisions, revision)
	s.ActiveFocus.ProfileRevisionID = revision.ID
	s.ActiveFocus.PrimaryAssetRefs = nil
	s.ActiveFocus.ExcludedAssetRefs = nil
	s.syncLegacyActiveProjection()
	return true
}

// ActiveProfile returns a copy of the active birth-data revision for read-only
// consumers. Mutation must go through MergeProfile to retain revision lineage.
func (s *SessionState) ActiveProfile() map[string]any {
	s.MigrateLegacyAssets()
	profile, ok := s.activeProfile()
	if !ok {
		return map[string]any{}
	}
	return cloneMap(profile.BirthData)
}

// StoreChart appends a new chart asset and selects it for the active subject.
// A tool result is never allowed to replace an older subject's chart in place.
func (s *SessionState) StoreChart(kind string, payload map[string]any, methodVersion string) AssetRef {
	s.MigrateLegacyAssets()
	if kind == AssetKindQimenCaseChart {
		caseID := strings.TrimSpace(s.ActiveFocus.CaseID)
		if caseID == "" {
			return AssetRef{}
		}
		return s.StoreChartForOwner(kind, AssetRef{Kind: "case", ID: caseID}, payload, methodVersion)
	}
	if kind == AssetKindQimenChart {
		caseID := strings.TrimSpace(s.ActiveFocus.CaseID)
		if caseID == "" {
			return AssetRef{}
		}
		// Keep old callers readable during migration, but never persist the legacy
		// kind or create a Case from a chart write.
		return s.StoreChartForOwner(AssetKindQimenCaseChart, AssetRef{Kind: "case", ID: caseID}, payload, methodVersion)
	}
	return s.StoreChartForOwner(kind, AssetRef{Kind: AssetKindProfileRevision, ID: s.ActiveFocus.ProfileRevisionID}, payload, methodVersion)
}

// StoreChartForOwner appends a chart under an explicit owner and selects it for
// the active subject. Qimen case charts require a non-empty case owner; the
// method never infers an owner from ActiveFocus or ProfileRevision.
func (s *SessionState) StoreChartForOwner(kind string, owner AssetRef, payload map[string]any, methodVersion string) AssetRef {
	if s == nil || owner.Kind == "" || owner.ID == "" {
		return AssetRef{}
	}
	if kind == AssetKindQimenCaseChart && owner.Kind != "case" {
		return AssetRef{}
	}
	s.MigrateLegacyAssets()
	subject := s.ActiveSubject()
	inputRefs := []AssetRef{}
	if owner.Kind == AssetKindProfileRevision {
		inputRefs = append(inputRefs, AssetRef{Kind: AssetKindProfileRevision, ID: owner.ID, Version: profileVersion(s, owner.ID)})
	}

	version := 1
	for _, asset := range s.Assets {
		if asset.Ref.Kind == kind && asset.OwnerKind == owner.Kind && asset.OwnerID == owner.ID && asset.Ref.Version >= version {
			version = asset.Ref.Version + 1
		}
	}
	copyPayload := cloneMap(payload)
	asset := DomainAsset{
		Ref:           AssetRef{Kind: kind, ID: fmt.Sprintf("%s-%s-v%d", kind, owner.ID, version), Version: version},
		OwnerKind:     owner.Kind,
		OwnerID:       owner.ID,
		SubjectIDs:    []string{subject.ID},
		InputRefs:     inputRefs,
		MethodVersion: methodVersion,
		CalendarRule:  stringValue(copyPayload["calendar_rule_version"]),
		PayloadHash:   payloadFingerprint(copyPayload),
		Payload:       copyPayload,
		CreatedAt:     time.Now().UTC(),
	}
	s.Assets = append(s.Assets, asset)
	s.clearExcludedAssets(kind)
	s.selectPrimaryAsset(asset.Ref)
	s.syncLegacyActiveProjection()
	return asset.Ref
}

// StoreInterpretation appends a reading that can only be reused with the exact
// active chart it cites. It intentionally does not become a primary chart
// selection, because it is derived narrative rather than deterministic input.
func (s *SessionState) StoreInterpretation(domain string, payload map[string]any) (AssetRef, bool) {
	s.MigrateLegacyAssets()
	chartKind := chartKindForDomain(domain)
	if chartKind == "" {
		return AssetRef{}, false
	}
	var chartRef AssetRef
	for _, ref := range s.ActiveFocus.PrimaryAssetRefs {
		if ref.Kind == chartKind {
			chartRef = ref
			break
		}
	}
	if chartRef.ID == "" {
		return AssetRef{}, false
	}
	subject := s.ActiveSubject()
	version := 1
	for _, asset := range s.Assets {
		if asset.Ref.Kind == AssetKindInterpretation && asset.OwnerID == chartRef.ID && asset.Ref.Version >= version {
			version = asset.Ref.Version + 1
		}
	}
	copyPayload := cloneMap(payload)
	asset := DomainAsset{
		Ref:           AssetRef{Kind: AssetKindInterpretation, ID: fmt.Sprintf("interpretation-%s-v%d", chartRef.ID, version), Version: version},
		OwnerKind:     "domain_asset",
		OwnerID:       chartRef.ID,
		SubjectIDs:    []string{subject.ID},
		InputRefs:     []AssetRef{chartRef},
		MethodVersion: "manager-followup-v1",
		PayloadHash:   payloadFingerprint(copyPayload),
		Payload:       copyPayload,
		CreatedAt:     time.Now().UTC(),
	}
	s.Assets = append(s.Assets, asset)
	return asset.Ref, true
}

// ActiveInterpretation returns the latest reading tied to the exact active
// chart for the requested domain. A different subject, profile revision or Case
// cannot satisfy this lookup.
func (s *SessionState) ActiveInterpretation(domain string) map[string]any {
	s.MigrateLegacyAssets()
	chartKind := chartKindForDomain(domain)
	if chartKind == "" {
		return nil
	}
	for _, chartRef := range s.ActiveFocus.PrimaryAssetRefs {
		if chartRef.Kind != chartKind {
			continue
		}
		for i := len(s.Assets) - 1; i >= 0; i-- {
			asset := s.Assets[i]
			if asset.Ref.Kind == AssetKindInterpretation && asset.OwnerID == chartRef.ID && stringValue(asset.Payload["domain"]) == domain {
				return asset.Payload
			}
		}
	}
	return nil
}

func chartKindForDomain(domain string) string {
	switch domain {
	case "bazi":
		return AssetKindBaziChart
	case "ziwei":
		return AssetKindZiweiChart
	case "qimen":
		return AssetKindQimenCaseChart
	default:
		return ""
	}
}

// InvalidateActiveChart removes only the current selection for a chart kind.
// The historical asset remains preserved for audit and a later explicit revisit.
func (s *SessionState) InvalidateActiveChart(kind string) {
	if s == nil {
		return
	}
	s.MigrateLegacyAssets()
	filtered := s.ActiveFocus.PrimaryAssetRefs[:0]
	for _, ref := range s.ActiveFocus.PrimaryAssetRefs {
		if ref.Kind != kind {
			filtered = append(filtered, ref)
			continue
		}
		if !s.isExcludedAsset(ref) {
			s.ActiveFocus.ExcludedAssetRefs = append(s.ActiveFocus.ExcludedAssetRefs, ref)
		}
	}
	s.ActiveFocus.PrimaryAssetRefs = filtered
	s.syncLegacyActiveProjection()
}

// StartCase selects an existing compatible active case or creates a new one.
// Event-chart callers should request a fresh case when the user starts a new
// question; natal-chart callers normally do not need a case at all.
func (s *SessionState) StartCase(domain, question string, fresh bool) Case {
	return s.StartCaseAt(domain, question, nil, fresh)
}

// StartCaseAt selects a compatible Case or creates one with an optional event
// time. A supplied event time prevents reusing a Case cast at another instant.
func (s *SessionState) StartCaseAt(domain, question string, eventTime *time.Time, fresh bool) Case {
	s.MigrateLegacyAssets()
	if !fresh && s.ActiveFocus.CaseID != "" {
		for _, item := range s.Cases {
			if item.ID == s.ActiveFocus.CaseID && item.Domain == domain {
				if eventTime == nil {
					return item
				}
				if item.EventTime == nil {
					item.EventTime = cloneTime(eventTime)
					s.replaceCase(item)
					return item
				}
				if item.EventTime.Equal(*eventTime) {
					return item
				}
			}
		}
	}
	s.ActiveFocus.CaseID = ""
	return s.ensureActiveCaseAt(domain, question, eventTime)
}

// QimenChartForCase returns the latest qimen chart whose owner is the exact
// supplied Case. It does not consult ActiveFocus or ProfileRevision fallbacks.
func (s *SessionState) QimenChartForCase(caseID string) map[string]any {
	if s == nil || strings.TrimSpace(caseID) == "" {
		return nil
	}
	for i := len(s.Assets) - 1; i >= 0; i-- {
		asset := s.Assets[i]
		if asset.OwnerKind != "case" || asset.OwnerID != caseID {
			continue
		}
		if asset.Ref.Kind != AssetKindQimenCaseChart && asset.Ref.Kind != AssetKindQimenChart {
			continue
		}
		return asset.Payload
	}
	return nil
}

// ActiveChart returns the exact active payload for one chart kind. It returns a
// mutable map only for existing runtime compatibility; new code should replace
// results through StoreChart after a deterministic tool call.
func (s *SessionState) ActiveChart(kind string) map[string]any {
	s.MigrateLegacyAssets()
	for _, ref := range s.ActiveFocus.PrimaryAssetRefs {
		if !sameChartKind(ref.Kind, kind) {
			continue
		}
		if asset, ok := s.assetByRef(ref); ok {
			return asset.Payload
		}
	}
	if isQimenChartKind(kind) {
		return nil
	}
	profileID := s.ActiveFocus.ProfileRevisionID
	for i := len(s.Assets) - 1; i >= 0; i-- {
		asset := s.Assets[i]
		if asset.Ref.Kind == kind && asset.OwnerID == profileID && !s.isExcludedAsset(asset.Ref) {
			s.selectPrimaryAsset(asset.Ref)
			return asset.Payload
		}
	}
	return nil
}

func isQimenChartKind(kind string) bool {
	return kind == AssetKindQimenChart || kind == AssetKindQimenCaseChart
}

func sameChartKind(left, right string) bool {
	return left == right || (isQimenChartKind(left) && isQimenChartKind(right))
}

// MigrateLegacyAssets imports pre-asset sessions once. Legacy fields remain a
// read projection during the transition, never the authority for new writes.
func (s *SessionState) MigrateLegacyAssets() {
	if s == nil || s.assetsMigrating {
		return
	}

	// StoreChart synchronizes legacy projections after each write. Preserve the
	// complete old snapshot first, otherwise importing bazi can erase a pending
	// ziwei/qimen projection before it has become an asset.
	legacyBazi := cloneMap(s.BaziResult)
	legacyZiwei := cloneMap(s.ZiWeiResult)
	legacyQimen := cloneMap(s.QimenResult)
	s.assetsMigrating = true
	defer func() {
		s.syncLegacyActiveProjection()
		s.assetsMigrating = false
	}()

	wasMigrated := s.assetsMigrated
	if !wasMigrated {
		s.assetsMigrated = true
	}
	if len(s.Subjects) == 0 {
		subject := s.EnsureSubject(firstNonEmptyState(s.Subject, "自己"))
		s.ActiveFocus.SubjectIDs = []string{subject.ID}
	}
	if len(s.ProfileRevisions) == 0 && len(s.Profile) > 0 {
		subject := s.ActiveSubject()
		revision := ProfileRevision{
			ID: "profile-" + subject.ID + "-v1", SubjectID: subject.ID, Version: 1,
			BirthData: cloneMap(s.Profile), Fingerprint: profileFingerprint(s.Profile), CreatedAt: time.Now().UTC(),
		}
		s.ProfileRevisions = append(s.ProfileRevisions, revision)
		s.ActiveFocus.ProfileRevisionID = revision.ID
	}
	s.importLegacyChartIfMissing(AssetKindBaziChart, legacyBazi, wasMigrated)
	s.importLegacyChartIfMissing(AssetKindZiweiChart, legacyZiwei, wasMigrated)
	s.importLegacyChartIfMissing(AssetKindQimenChart, legacyQimen, wasMigrated)
}

func (s *SessionState) importLegacyChartIfMissing(kind string, payload map[string]any, wasMigrated bool) {
	if len(payload) == 0 || s.activeChartWithoutMigration(kind) != nil {
		return
	}
	// Once typed assets exist, an unchanged legacy map is merely the old active
	// projection. Re-import only an externally replaced map, which keeps legacy
	// test fixtures/readers working without letting a stale projection create a
	// chart for a corrected profile or newly selected person.
	if wasMigrated && s.legacyProjectionHash(kind) == payloadFingerprint(payload) {
		return
	}
	s.StoreChart(kind, payload, "legacy")
}

func (s *SessionState) activeChartWithoutMigration(kind string) map[string]any {
	for _, ref := range s.ActiveFocus.PrimaryAssetRefs {
		if ref.Kind != kind {
			continue
		}
		if asset, ok := s.assetByRef(ref); ok {
			return asset.Payload
		}
	}
	profileID := s.ActiveFocus.ProfileRevisionID
	for i := len(s.Assets) - 1; i >= 0; i-- {
		asset := s.Assets[i]
		if asset.Ref.Kind == kind && asset.OwnerID == profileID {
			return asset.Payload
		}
	}
	return nil
}

func (s *SessionState) activeProfile() (ProfileRevision, bool) {
	for _, profile := range s.ProfileRevisions {
		if profile.ID == s.ActiveFocus.ProfileRevisionID {
			return profile, true
		}
	}
	return ProfileRevision{}, false
}

func (s *SessionState) latestProfileForSubject(subjectID string) (ProfileRevision, bool) {
	var latest ProfileRevision
	found := false
	for _, profile := range s.ProfileRevisions {
		if profile.SubjectID == subjectID && (!found || profile.Version > latest.Version) {
			latest, found = profile, true
		}
	}
	return latest, found
}

func (s *SessionState) ensureActiveCase(domain, question string) Case {
	return s.ensureActiveCaseAt(domain, question, nil)
}

func (s *SessionState) ensureActiveCaseAt(domain, question string, eventTime *time.Time) Case {
	if s.ActiveFocus.CaseID != "" {
		for _, item := range s.Cases {
			if item.ID == s.ActiveFocus.CaseID {
				return item
			}
		}
	}
	subject := s.ActiveSubject()
	item := Case{ID: fmt.Sprintf("case-%d", len(s.Cases)+1), Domain: domain, Question: question, SubjectIDs: []string{subject.ID}, EventTime: cloneTime(eventTime), CreatedAt: time.Now().UTC()}
	s.Cases = append(s.Cases, item)
	s.ActiveFocus.CaseID = item.ID
	return item
}

func (s *SessionState) replaceCase(updated Case) {
	for i := range s.Cases {
		if s.Cases[i].ID == updated.ID {
			s.Cases[i] = updated
			return
		}
	}
}

func (s *SessionState) selectPrimaryAsset(ref AssetRef) {
	for i, existing := range s.ActiveFocus.PrimaryAssetRefs {
		if sameChartKind(existing.Kind, ref.Kind) {
			s.ActiveFocus.PrimaryAssetRefs[i] = ref
			return
		}
	}
	s.ActiveFocus.PrimaryAssetRefs = append(s.ActiveFocus.PrimaryAssetRefs, ref)
}

func (s *SessionState) isExcludedAsset(ref AssetRef) bool {
	for _, excluded := range s.ActiveFocus.ExcludedAssetRefs {
		if excluded == ref {
			return true
		}
	}
	return false
}

func (s *SessionState) clearExcludedAssets(kind string) {
	filtered := s.ActiveFocus.ExcludedAssetRefs[:0]
	for _, ref := range s.ActiveFocus.ExcludedAssetRefs {
		if !sameChartKind(ref.Kind, kind) {
			filtered = append(filtered, ref)
		}
	}
	s.ActiveFocus.ExcludedAssetRefs = filtered
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (s *SessionState) assetByRef(ref AssetRef) (DomainAsset, bool) {
	for _, asset := range s.Assets {
		if asset.Ref == ref {
			return asset, true
		}
	}
	return DomainAsset{}, false
}

func (s *SessionState) syncLegacyActiveProjection() {
	if s == nil {
		return
	}
	if profile, ok := s.activeProfile(); ok {
		s.Profile = cloneMap(profile.BirthData)
	} else if s.Profile == nil {
		s.Profile = map[string]any{}
	}
	s.BaziResult = s.ActiveChart(AssetKindBaziChart)
	s.ZiWeiResult = s.ActiveChart(AssetKindZiweiChart)
	s.QimenResult = s.ActiveChart(AssetKindQimenChart)
	if s.legacyProjectionHashes == nil {
		s.legacyProjectionHashes = make(map[string]string, 3)
	}
	s.legacyProjectionHashes[AssetKindBaziChart] = payloadFingerprint(s.BaziResult)
	s.legacyProjectionHashes[AssetKindZiweiChart] = payloadFingerprint(s.ZiWeiResult)
	s.legacyProjectionHashes[AssetKindQimenChart] = payloadFingerprint(s.QimenResult)
}

func (s *SessionState) legacyProjectionHash(kind string) string {
	if s == nil || s.legacyProjectionHashes == nil {
		return ""
	}
	return s.legacyProjectionHashes[kind]
}

func profileVersion(s *SessionState, profileID string) int {
	for _, p := range s.ProfileRevisions {
		if p.ID == profileID {
			return p.Version
		}
	}
	return 0
}
func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneAssetValue(v)
	}
	return out
}

func cloneAssetValue(value any) any {
	switch item := value.(type) {
	case map[string]any:
		return cloneMap(item)
	case []any:
		out := make([]any, len(item))
		for i, entry := range item {
			out[i] = cloneAssetValue(entry)
		}
		return out
	case []string:
		return append([]string(nil), item...)
	default:
		return item
	}
}
func cloneProfileRevisions(in []ProfileRevision) []ProfileRevision {
	out := make([]ProfileRevision, len(in))
	for i, v := range in {
		out[i] = v
		out[i].BirthData = cloneMap(v.BirthData)
	}
	return out
}
func cloneCases(in []Case) []Case {
	out := make([]Case, len(in))
	for i, v := range in {
		out[i] = v
		out[i].SubjectIDs = append([]string(nil), v.SubjectIDs...)
		if v.EventTime != nil {
			t := *v.EventTime
			out[i].EventTime = &t
		}
	}
	return out
}
func cloneDomainAssets(in []DomainAsset) []DomainAsset {
	out := make([]DomainAsset, len(in))
	for i, v := range in {
		out[i] = v
		out[i].SubjectIDs = append([]string(nil), v.SubjectIDs...)
		out[i].InputRefs = append([]AssetRef(nil), v.InputRefs...)
		out[i].Payload = cloneMap(v.Payload)
		if v.EffectiveTime != nil {
			t := *v.EffectiveTime
			out[i].EffectiveTime = &t
		}
	}
	return out
}
func cloneActiveFocus(in ActiveFocus) ActiveFocus {
	return ActiveFocus{
		SubjectIDs:        append([]string(nil), in.SubjectIDs...),
		ProfileRevisionID: in.ProfileRevisionID,
		CaseID:            in.CaseID,
		PrimaryAssetRefs:  append([]AssetRef(nil), in.PrimaryAssetRefs...),
		ExcludedAssetRefs: append([]AssetRef(nil), in.ExcludedAssetRefs...),
	}
}
func stringValue(v any) string { value, _ := v.(string); return value }
func payloadFingerprint(payload map[string]any) string {
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func profileFingerprint(profile map[string]any) string { return payloadFingerprint(profile) }
func firstNonEmptyState(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
