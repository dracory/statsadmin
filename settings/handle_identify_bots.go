package settings

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"

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

// scanBatchSize is the number of records fetched per query during the scan.
const scanBatchSize = 1000

// userAgentPattern is the reason pattern label for user-agent-based bot
// detection. It is not a path substring — it indicates the IP was flagged
// because its user agent self-identified as a bot.
const userAgentPattern = "user-agent:bot"

// autoAddThreshold is the minimum hit count at which an IP is considered a
// bot regardless of evidence type. Rationale: no human visitor requests
// bot-only files (robots.txt, ads.txt, sitemap.xml, etc.) more than once.
const autoAddThreshold = 2

func (controller *settingsController) handleIdentifyBots(w http.ResponseWriter, r *http.Request) string {
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

	// Get existing bot IPs from the visitor table (bot='yes' column).
	existingBots := getExistingBotIPs(ctx, store)
	allCandidates := collectBotCandidates(ctx, store, existingBots)

	// Separate high-confidence candidates from unsure ones.
	// An IP is auto-added when ANY of:
	//   - its user agent self-identified as a bot (IsBot match), OR
	//   - it hit a malicious pattern (.php, .env, wp-admin, etc.) — a single
	//     hit to these is unambiguous since no human or legitimate bot ever
	//     requests them, OR
	//   - it has 2+ hits to bot-only files (no human visits these twice).
	// Single-hit path-only IPs on legitimate bot files go to "needs review".
	var autoAdded []string
	unsure := map[string]*botCandidate{}
	for ip, c := range allCandidates {
		if c.hasUserAgentEvidence() || c.hasMaliciousEvidence() || c.Hits >= autoAddThreshold {
			autoAdded = append(autoAdded, ip)
		} else {
			unsure[ip] = c
		}
	}

	// Set bot='yes' on all visitor records for auto-added IPs.
	// Track which IPs actually succeeded so we don't report failures as auto-added.
	var successfulAutoAdded []string
	for _, ip := range autoAdded {
		_, err := shared.FlagIPVisitorsAsBot(ctx, store, ip)
		if err != nil {
			slog.Error("statsadmin settings: failed to update visitor bot flag", "ip", ip, "error", err)
			// Move failed IP back to unsure.
			if c, ok := allCandidates[ip]; ok {
				unsure[ip] = c
			}
			continue
		}
		successfulAutoAdded = append(successfulAutoAdded, ip)
	}
	autoAdded = successfulAutoAdded

	// Reload bot IPs to include the auto-added ones.
	updatedBots := getExistingBotIPs(ctx, store)

	response := buildCandidateResponse(unsure, updatedBots)
	response[FieldAutoAdded] = autoAdded
	response[FieldAutoAddedCount] = len(autoAdded)
	api.Respond(w, r, api.SuccessWithData("Bot candidates identified", response))
	return ""
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

// collectBotCandidates scans the stats store for bot candidates in a single
// pass over all visitor records, fetching them in batches. For each record it
// checks the path against every bot-page pattern (case-insensitive substring
// match) and the user agent against botdetect.IsBot(). One pass avoids
// running a full table scan per pattern, which previously made the endpoint
// time out on large tables. IPs already in the bot list are skipped.
func collectBotCandidates(ctx context.Context, store statsstore.StoreInterface, existingBots []string) map[string]*botCandidate {
	ipAgg := map[string]*botCandidate{}
	skipSet := map[string]bool{}
	for _, ip := range existingBots {
		skipSet[ip] = true
	}

	offset := 0
	for {
		query := statsstore.VisitorQuery().
			SetOrderBy(statsstore.COLUMN_CREATED_AT).
			SetSortOrder("ASC").
			SetLimit(scanBatchSize).
			SetOffset(offset)

		visitors, err := store.VisitorList(ctx, query)
		if err != nil || len(visitors) == 0 {
			return ipAgg
		}

		for _, v := range visitors {
			if skipSet[v.GetIpAddress()] {
				continue
			}

			// Path scan: case-insensitive substring match against all
			// bot-page patterns (same matching as handle_load_bots.go).
			pathLower := strings.ToLower(v.GetPath())
			for _, pattern := range botPagePatterns {
				if strings.Contains(pathLower, pattern) {
					mergeVisitorIntoCandidates(ipAgg, v, pattern)
				}
			}

			// User-agent scan: check self-identifying bots (e.g.
			// "SemrushBot", "Googlebot", "curl") that don't necessarily
			// visit bot pages.
			ua := v.GetUserAgent()
			if ua != "" && botdetect.IsBot(ua) {
				mergeVisitorIntoCandidates(ipAgg, v, userAgentPattern)
			}
		}

		if len(visitors) < scanBatchSize {
			return ipAgg
		}
		offset += scanBatchSize
	}
}

// mergeVisitorIntoCandidates adds a single visitor record to the aggregation
// map, incrementing the hit count and tracking per-pattern evidence.
func mergeVisitorIntoCandidates(ipAgg map[string]*botCandidate, v statsstore.VisitorInterface, pattern string) {
	ip := v.GetIpAddress()
	if ip == "" {
		return
	}

	cand, exists := ipAgg[ip]
	if !exists {
		cand = &botCandidate{IP: ip, Reasons: map[string]*botReason{}}
		ipAgg[ip] = cand
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
