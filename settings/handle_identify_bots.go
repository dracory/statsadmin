package settings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/dracory/statsadmin/shared"

	"github.com/dracory/api"
	"github.com/dracory/botdetect"
	"github.com/dracory/statsstore"
)

// stackDependentMaliciousPatterns are path patterns that are malicious on a
// Go site but could be legitimate on other tech stacks (e.g. .php on a PHP
// site, wp-admin on WordPress). This is a Go site, so all of these are
// treated as malicious. Universal malicious patterns (.env, .git/, .svn/,
// .htpasswd, shell.php) come from botdetect.MaliciousPathPatterns().
var stackDependentMaliciousPatterns = []string{
	botPagePatternPhp,
	botPagePatternPy,
	botPagePatternAsp,
	botPagePatternAspx,
	botPagePatternJsp,
	botPagePatternCgi,
	botPagePatternPl,
	botPagePatternRb,
	botPagePatternWp,
	botPagePatternWordPress,
	botPagePatternWpAdmin,
	botPagePatternWpLogin,
	botPagePatternWpIncludes,
	botPagePatternWpContent,
	botPagePatternWlwManifest,
	botPagePatternXmlRpc,
	botPagePatternConfig,
	botPagePatternAdminer,
}

// consumerSpecificBotPathPatterns are bot-file patterns specific to this
// consumer that are too broad for the universal statsstore list (e.g.
// "google" matches any Google verification file but also any path containing
// "google").
var consumerSpecificBotPathPatterns = []string{
	botPagePatternGoogleVerify,
	botPagePatternWellKnown,
}

// allMaliciousPatterns combines universal malicious patterns from statsstore
// with stack-dependent patterns for this Go site. A single hit to any of
// these is sufficient evidence to auto-add the IP as a bot.
var allMaliciousPatterns = func() map[string]bool {
	m := map[string]bool{}
	for _, p := range botdetect.MaliciousPathPatterns() {
		m[strings.ToLower(p)] = true
	}
	for _, p := range stackDependentMaliciousPatterns {
		m[strings.ToLower(p)] = true
	}
	return m
}()

// botPagePatterns lists all path substrings to scan for. It combines:
//   - universal bot file patterns from statsstore (robots.txt, ads.txt, etc.)
//   - universal malicious patterns from statsstore (.env, .git/, etc.)
//   - consumer-specific bot file patterns (google verification)
//   - stack-dependent malicious patterns for this Go site (.php, wp-admin, etc.)
var botPagePatterns = func() []string {
	patterns := botdetect.BotPathPatterns()
	patterns = append(patterns, botdetect.MaliciousPathPatterns()...)
	patterns = append(patterns, consumerSpecificBotPathPatterns...)
	patterns = append(patterns, stackDependentMaliciousPatterns...)
	for i, p := range patterns {
		patterns[i] = strings.ToLower(p)
	}
	return patterns
}()

// scanBatchSize is the number of visitor records processed per step request.
const scanBatchSize = 1000

// flagBatchSize is the number of candidate IPs flagged per step request once
// the scan finishes.
const flagBatchSize = 10

// maxStoredCandidates caps the candidate map kept in the settings table so
// the persisted state stays bounded on very noisy logs.
const maxStoredCandidates = 20000

// userAgentPattern is the reason pattern label for user-agent-based bot
// detection. It is not a path substring — it indicates the IP was flagged
// because its user agent self-identified as a bot.
const userAgentPattern = "user-agent:bot"

// autoAddThreshold is the minimum hit count at which an IP is considered a
// bot regardless of evidence type. Rationale: no human visitor requests
// bot-only files (robots.txt, ads.txt, sitemap.xml, etc.) more than once.
const autoAddThreshold = 2

// settingKeyBotScanState is the settings-table key where the in-progress
// scan state is persisted between step requests.
const settingKeyBotScanState = "statsadmin.botscan.state"

// settingKeyKnownBotIPs is the settings-table key where the last known bot
// IP list is cached after a scan completes, so consumers do not need to
// re-query the visitors table for it.
const settingKeyKnownBotIPs = "statsadmin.bot_ips"

// Scan phases persisted in the state record.
const (
	botScanPhaseScan = "scan"
	botScanPhaseFlag = "flag"
	botScanPhaseDone = "done"
)

// identifyBootID is generated once per process. Because a server restart
// produces a new boot ID, any scan state persisted by a previous process is
// treated as abandoned: step requests report "restarted" and the frontend
// starts a fresh identification from scratch.
var identifyBootID = newScanRunID()

// botScanState is the resumable scan state persisted in the settings table
// between step requests.
type botScanState struct {
	RunID string `json:"run_id"`
	// BootID ties the state to the process that created it; a restart
	// invalidates it (see identifyBootID).
	BootID string `json:"boot_id"`
	Phase  string `json:"phase"`
	// CursorTS / CursorIDs implement keyset pagination: CursorTS is the
	// created_at of the last processed record and CursorIDs are the IDs of
	// already-processed records sharing that exact timestamp (created_at is
	// not unique, so the ID list disambiguates ties).
	CursorTS   string                   `json:"cursor_ts"`
	CursorIDs  []string                 `json:"cursor_ids"`
	Scanned    int                      `json:"scanned"`
	Total      int64                    `json:"total"`
	SkipIPs    []string                 `json:"skip_ips"`
	Candidates map[string]*botCandidate `json:"candidates"`
	// Truncated counts candidate IPs dropped once maxStoredCandidates was
	// reached.
	Truncated int      `json:"truncated"`
	FlagQueue []string `json:"flag_queue"`
	FlagIndex int      `json:"flag_index"`
	AutoAdded []string `json:"auto_added"`
	// Result is the final payload, populated when Phase reaches "done".
	Result map[string]any `json:"result,omitempty"`
}

// newScanRunID returns a random identifier for a scan run or process boot.
func newScanRunID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}

func (controller *settingsController) handleIdentifyBotsStart(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	ctx := r.Context()

	total, err := store.VisitorCount(ctx, statsstore.VisitorQuery())
	if err != nil {
		slog.Error("statsadmin settings: failed to count visitors", "error", err)
		api.Respond(w, r, api.Error("Failed to count visitor records"))
		return ""
	}

	state := &botScanState{
		RunID:      newScanRunID(),
		BootID:     identifyBootID,
		Phase:      botScanPhaseScan,
		Total:      total,
		SkipIPs:    getExistingBotIPs(ctx, store),
		Candidates: map[string]*botCandidate{},
	}
	if err := saveBotScanState(ctx, store, state); err != nil {
		slog.Error("statsadmin settings: failed to save scan state", "error", err)
		api.Respond(w, r, api.Error("Failed to save scan state"))
		return ""
	}

	api.Respond(w, r, api.SuccessWithData("Bot identification started", map[string]any{
		FieldRunID:   state.RunID,
		FieldPhase:   state.Phase,
		FieldScanned: 0,
		FieldTotal:   total,
		FieldPercent: 0,
		FieldDone:    false,
	}))
	return ""
}

func (controller *settingsController) handleIdentifyBotsStep(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	var reqBody struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Respond(w, r, api.Error("Invalid request body"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	ctx := r.Context()

	state, err := loadBotScanState(ctx, store)
	if err != nil {
		slog.Error("statsadmin settings: failed to load scan state", "error", err)
		api.Respond(w, r, api.Error("Failed to load scan state"))
		return ""
	}
	if state == nil {
		respondScanRestarted(w, r, "No bot identification in progress")
		return ""
	}
	if state.BootID != identifyBootID {
		respondScanRestarted(w, r, "Server restarted; please run identification again")
		return ""
	}
	if state.RunID != reqBody.RunID {
		respondScanRestarted(w, r, "Scan was restarted; please use the latest run")
		return ""
	}

	switch state.Phase {
	case botScanPhaseScan:
		controller.stepScan(ctx, store, state)
	case botScanPhaseFlag:
		controller.stepFlag(ctx, store, state)
	case botScanPhaseDone:
		// Nothing to do; final result is reported below.
	}

	if err := saveBotScanState(ctx, store, state); err != nil {
		slog.Error("statsadmin settings: failed to save scan state", "error", err)
		api.Respond(w, r, api.Error("Failed to save scan state"))
		return ""
	}

	data := map[string]any{
		FieldRunID:   state.RunID,
		FieldPhase:   state.Phase,
		FieldScanned: state.Scanned,
		FieldTotal:   state.Total,
		FieldPercent: state.percent(),
		FieldDone:    state.Phase == botScanPhaseDone,
		FieldFlagged: len(state.AutoAdded),
	}
	if state.Truncated > 0 {
		data["candidates_truncated"] = state.Truncated
	}
	if state.Phase == botScanPhaseDone && state.Result != nil {
		data[FieldResult] = state.Result
	}
	api.Respond(w, r, api.SuccessWithData("Bot identification step completed", data))
	return ""
}

// respondScanRestarted tells the frontend the persisted state is stale and
// a new run must be started from scratch.
func respondScanRestarted(w http.ResponseWriter, r *http.Request, message string) {
	api.Respond(w, r, api.ErrorWithData(message, map[string]any{
		FieldRestarted: true,
	}))
}

// stepScan processes up to scanBatchSize visitor records using keyset
// pagination over created_at. When the table is exhausted it transitions the
// state to the flag phase.
func (controller *settingsController) stepScan(ctx context.Context, store statsstore.StoreInterface, state *botScanState) {
	skipSet := map[string]bool{}
	for _, ip := range state.SkipIPs {
		skipSet[ip] = true
	}
	if state.Candidates == nil {
		state.Candidates = map[string]*botCandidate{}
	}
	seenSet := map[string]bool{}
	for _, id := range state.CursorIDs {
		seenSet[id] = true
	}

	processed := 0
	offset := 0
	exhausted := false
	for processed < scanBatchSize {
		query := statsstore.VisitorQuery().
			SetOrderBy(statsstore.COLUMN_CREATED_AT).
			SetSortOrder("ASC").
			SetLimit(scanBatchSize).
			SetOffset(offset)
		// Empty cursor means "from the beginning"; SetCreatedAtGte rejects
		// empty strings, so only apply it once a cursor exists.
		if state.CursorTS != "" {
			query.SetCreatedAtGte(state.CursorTS)
		}
		visitors, err := store.VisitorList(ctx, query)
		if err != nil {
			slog.Error("statsadmin settings: scan batch query failed", "error", err)
			exhausted = true
			break
		}
		if len(visitors) == 0 {
			exhausted = true
			break
		}
		offset += len(visitors)

		for _, v := range visitors {
			if processed >= scanBatchSize {
				break
			}
			ts := v.GetCreatedAt()
			id := v.GetID()
			switch {
			case ts > state.CursorTS:
				state.CursorTS = ts
				seenSet = map[string]bool{id: true}
			case seenSet[id]:
				continue
			default:
				seenSet[id] = true
			}
			controller.scanVisitor(state, skipSet, v)
			state.Scanned++
			processed++
		}

		if len(visitors) < scanBatchSize {
			exhausted = true
			break
		}
	}

	state.CursorIDs = make([]string, 0, len(seenSet))
	for id := range seenSet {
		state.CursorIDs = append(state.CursorIDs, id)
	}

	if !exhausted {
		return
	}

	// Scan finished — split candidates into an auto-add queue and the
	// "needs review" set, then move to the flag phase. An IP is auto-added
	// when ANY of:
	//   - its user agent self-identified as a bot, OR
	//   - it hit a malicious pattern (.php, .env, wp-admin, etc.), OR
	//   - it has 2+ hits to bot-only files.
	// Single-hit path-only IPs stay in Candidates for manual review.
	var queue []string
	for ip, c := range state.Candidates {
		if c.hasUserAgentEvidence() || c.hasMaliciousEvidence() || c.Hits >= autoAddThreshold {
			queue = append(queue, ip)
		}
	}
	slices.Sort(queue)
	for _, ip := range queue {
		delete(state.Candidates, ip)
	}
	state.FlagQueue = queue
	state.FlagIndex = 0
	state.Phase = botScanPhaseFlag
	if len(queue) == 0 {
		controller.finishScan(ctx, store, state)
	}
}

// scanVisitor applies the path-pattern and user-agent checks to a single
// visitor record and merges the evidence into the candidate map.
func (controller *settingsController) scanVisitor(state *botScanState, skipSet map[string]bool, v statsstore.VisitorInterface) {
	ip := v.GetIpAddress()
	if ip == "" || skipSet[ip] {
		return
	}

	pathLower := strings.ToLower(v.GetPath())
	for _, pattern := range botPagePatterns {
		if strings.Contains(pathLower, pattern) {
			mergeVisitorIntoCandidates(state, v, pattern)
		}
	}

	ua := v.GetUserAgent()
	if ua != "" && botdetect.IsBot(ua) {
		mergeVisitorIntoCandidates(state, v, userAgentPattern)
	}
}

// stepFlag flags up to flagBatchSize queued IPs per request. When the queue
// is drained it transitions the state to done.
func (controller *settingsController) stepFlag(ctx context.Context, store statsstore.StoreInterface, state *botScanState) {
	end := state.FlagIndex + flagBatchSize
	if end > len(state.FlagQueue) {
		end = len(state.FlagQueue)
	}
	for _, ip := range state.FlagQueue[state.FlagIndex:end] {
		// Recreate a minimal candidate so a failed flag can be moved back
		// to the review list with its evidence intact when available.
		if _, err := shared.FlagIPVisitorsAsBot(ctx, store, ip); err != nil {
			slog.Error("statsadmin settings: failed to flag IP as bot", "ip", ip, "error", err)
			state.Candidates[ip] = &botCandidate{IP: ip, Reasons: map[string]*botReason{}}
			continue
		}
		state.AutoAdded = append(state.AutoAdded, ip)
	}
	state.FlagIndex = end
	if state.FlagIndex >= len(state.FlagQueue) {
		controller.finishScan(ctx, store, state)
	}
}

// finishScan marks the run done, builds the final result payload, and
// caches the updated known-bot IP list in the settings table.
func (controller *settingsController) finishScan(ctx context.Context, store statsstore.StoreInterface, state *botScanState) {
	knownSet := map[string]bool{}
	for _, ip := range state.SkipIPs {
		knownSet[ip] = true
	}
	for _, ip := range state.AutoAdded {
		knownSet[ip] = true
	}
	known := make([]string, 0, len(knownSet))
	for ip := range knownSet {
		known = append(known, ip)
	}
	slices.Sort(known)
	if data, err := json.Marshal(known); err == nil {
		if err := store.SettingSet(ctx, settingKeyKnownBotIPs, string(data)); err != nil {
			slog.Error("statsadmin settings: failed to cache bot IP list", "error", err)
		}
	}

	response := buildCandidateResponse(state.Candidates, known)
	response[FieldAutoAdded] = state.AutoAdded
	response[FieldAutoAddedCount] = len(state.AutoAdded)
	state.Result = response
	state.Phase = botScanPhaseDone
}

// percent reports scan progress. Once the scan phase ends the scan itself
// is complete, so flag/done report 100.
func (s *botScanState) percent() int {
	if s.Phase != botScanPhaseScan || s.Total <= 0 {
		return 100
	}
	p := int(int64(s.Scanned) * 100 / s.Total)
	if p > 100 {
		p = 100
	}
	return p
}

// loadBotScanState reads the persisted scan state. Returns nil when no scan
// has been started.
func loadBotScanState(ctx context.Context, store statsstore.StoreInterface) (*botScanState, error) {
	raw, err := store.SettingGet(ctx, settingKeyBotScanState)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var state botScanState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveBotScanState(ctx context.Context, store statsstore.StoreInterface, state *botScanState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return store.SettingSet(ctx, settingKeyBotScanState, string(data))
}

// getExistingBotIPs returns the set of IPs that already have bot='yes' on at
// least one visitor record.
func getExistingBotIPs(ctx context.Context, store statsstore.StoreInterface) []string {
	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetBot(statsstore.VALUE_YES).
		SetLimit(100000))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var ips []string
	for _, v := range visitors {
		ip := v.GetIpAddress()
		if ip != "" && !seen[ip] {
			seen[ip] = true
			ips = append(ips, ip)
		}
	}
	return ips
}

// mergeVisitorIntoCandidates adds a single visitor record to the aggregation
// map, incrementing the hit count and tracking per-pattern evidence.
func mergeVisitorIntoCandidates(state *botScanState, v statsstore.VisitorInterface, pattern string) {
	ip := v.GetIpAddress()
	if ip == "" {
		return
	}

	cand, exists := state.Candidates[ip]
	if !exists {
		if len(state.Candidates) >= maxStoredCandidates {
			state.Truncated++
			return
		}
		cand = &botCandidate{IP: ip, Reasons: map[string]*botReason{}}
		state.Candidates[ip] = cand
	}
	cand.Hits++

	// Track unique user agents (cap at 5).
	ua := v.GetUserAgent()
	if ua != "" && !slices.Contains(cand.UserAgents, ua) && len(cand.UserAgents) < 5 {
		cand.UserAgents = append(cand.UserAgents, ua)
	}

	reason, ok := cand.Reasons[pattern]
	if !ok {
		reason = &botReason{Pattern: pattern}
		cand.Reasons[pattern] = reason
	}
	reason.Hits++

	// Track sample paths for this pattern (cap at 3).
	if len(reason.Paths) < 3 {
		path := v.GetPath()
		if !slices.Contains(reason.Paths, path) {
			reason.Paths = append(reason.Paths, path)
		}
	}
}

// buildCandidateResponse converts the candidate map into a sorted slice of
// response objects, marking IPs that are already in the bot list.
func buildCandidateResponse(ipAgg map[string]*botCandidate, existingBots []string) map[string]any {
	existingSet := map[string]bool{}
	for _, ip := range existingBots {
		existingSet[ip] = true
	}

	candidates := make([]map[string]any, 0, len(ipAgg))
	for _, c := range ipAgg {
		candidates = append(candidates, map[string]any{
			FieldCandidateIP:    c.IP,
			FieldCandidateHits:  c.Hits,
			FieldCandidatePaths: c.samplePaths(),
			FieldCandidateUAs:   c.UserAgents,
			FieldReasons:        c.reasonsList(),
			FieldAlreadyListed:  existingSet[c.IP],
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		hi, _ := candidates[i][FieldCandidateHits].(int)
		hj, _ := candidates[j][FieldCandidateHits].(int)
		if hi != hj {
			return hi > hj
		}
		pi, _ := candidates[i][FieldCandidateIP].(string)
		pj, _ := candidates[j][FieldCandidateIP].(string)
		return pi < pj
	})

	return map[string]any{
		FieldCandidates: candidates,
		FieldTotal:      len(candidates),
	}
}

// botCandidate is the internal aggregation struct used during the scan.
type botCandidate struct {
	IP         string
	Hits       int
	Reasons    map[string]*botReason
	UserAgents []string // unique user agents seen for this IP (up to 5)
}

// hasPathEvidence reports whether the candidate has any reason that is not
// the user-agent pattern (i.e. it hit a bot-page path).
func (c *botCandidate) hasPathEvidence() bool {
	for key := range c.Reasons {
		if key != userAgentPattern {
			return true
		}
	}
	return false
}

// hasUserAgentEvidence reports whether the candidate was flagged by the
// user-agent scan (i.e. its UA self-identified as a bot).
func (c *botCandidate) hasUserAgentEvidence() bool {
	_, ok := c.Reasons[userAgentPattern]
	return ok
}

// hasMaliciousEvidence reports whether the candidate hit any vulnerability-
// scan pattern (e.g. .php, .env, wp-admin). A single hit to any of these
// is sufficient proof of a malicious bot — no human or legitimate crawler
// ever requests them on a Go site.
func (c *botCandidate) hasMaliciousEvidence() bool {
	for key := range c.Reasons {
		if allMaliciousPatterns[key] {
			return true
		}
	}
	return false
}

// botReason tracks per-pattern evidence for a single candidate.
type botReason struct {
	Pattern string
	Hits    int
	Paths   []string
}

// samplePaths returns up to 20 unique paths across all reasons.
func (c *botCandidate) samplePaths() []string {
	seen := map[string]bool{}
	var paths []string
	for _, r := range c.Reasons {
		for _, p := range r.Paths {
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
				if len(paths) >= 20 {
					return paths
				}
			}
		}
	}
	return paths
}

// reasonsList converts the reason map into a sorted slice for the response.
func (c *botCandidate) reasonsList() []map[string]any {
	reasons := make([]map[string]any, 0, len(c.Reasons))
	for _, r := range c.Reasons {
		reasons = append(reasons, map[string]any{
			"pattern": r.Pattern,
			"hits":    r.Hits,
			"paths":   r.Paths,
		})
	}
	sort.Slice(reasons, func(i, j int) bool {
		pi, _ := reasons[i]["pattern"].(string)
		pj, _ := reasons[j]["pattern"].(string)
		return pi < pj
	})
	return reasons
}
