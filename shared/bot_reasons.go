package shared

import (
	"sort"
	"strings"

	"github.com/dracory/statsstore"
)

// BotReasonDisplay is a single reason an IP was flagged as a bot, formatted
// for display in the admin UI. It mirrors the JSON shape expected by the
// frontend (pattern, hits, sample paths).
type BotReasonDisplay struct {
	Pattern string   `json:"pattern"`
	Hits    int      `json:"hits"`
	Paths   []string `json:"paths"`
}

// Reason pattern labels for signals that are not path substrings. These are
// shown in the UI as the reason "pattern" so the user understands which
// heuristic triggered the bot flag.
const (
	// BotReasonUserAgent indicates the visitor's User-Agent string
	// self-identified as a bot (matched statsstore.IsBot).
	BotReasonUserAgent = "user-agent:bot"

	// BotReasonDataCenterIP indicates the visitor's IP falls within a known
	// data-center CIDR range (matched statsstore.IsDataCenterIP).
	BotReasonDataCenterIP = "datacenter-ip"

	// BotReasonReferrerSpam indicates the visitor's referrer matched a known
	// referrer-spam domain (matched statsstore.IsReferrerSpam).
	BotReasonReferrerSpam = "referrer-spam"
)

// ComputeBotReasons derives the bot reasons for a set of visitor records
// belonging to a single IP. It checks all signals used at ingestion time
// (see store.VisitorRegister): user-agent self-identification, data-center
// IP, referrer spam, bot-only paths, and malicious paths. extraPathPatterns
// is an optional list of consumer-specific path substrings to also check
// (e.g. ".php", "/wp-admin"); pass nil to check only the universal
// statsstore patterns.
//
// Universal bot/malicious path patterns use the same precise matching as
// statsstore.IsBotPath / statsstore.IsMaliciousPath (suffix on last path
// segment for file patterns, segment match for directory patterns). Extra
// consumer patterns use case-insensitive substring matching, consistent
// with the identify-bots scan. The returned slice is sorted by pattern for
// stable display.
func ComputeBotReasons(visitors []statsstore.VisitorInterface, extraPathPatterns []string) []BotReasonDisplay {
	type reasonAgg struct {
		Pattern string
		Hits    int
		Paths   []string
		seen    map[string]bool
	}

	reasons := map[string]*reasonAgg{}

	addReason := func(pattern, path string) {
		r, ok := reasons[pattern]
		if !ok {
			r = &reasonAgg{Pattern: pattern, seen: map[string]bool{}}
			reasons[pattern] = r
		}
		r.Hits++
		if path != "" && !r.seen[path] && len(r.Paths) < 3 {
			r.Paths = append(r.Paths, path)
			r.seen[path] = true
		}
	}

	botPatterns := statsstore.BotPathPatterns()
	maliciousPatterns := statsstore.MaliciousPathPatterns()

	// Pre-lowercase extra patterns once for case-insensitive substring
	// matching (matching the identify-bots scan behavior).
	extraLower := make([]string, len(extraPathPatterns))
	for i, p := range extraPathPatterns {
		extraLower[i] = strings.ToLower(p)
	}

	// dataCenterIP is the same for all visitors from the same IP, so only
	// record it once. We still iterate all visitors for the other signals.
	dataCenterChecked := false

	for _, v := range visitors {
		ua := v.GetUserAgent()
		ip := v.GetIpAddress()
		referrer := v.GetUserReferrer()
		path := v.GetPath()

		// User-agent self-identification (e.g. "Googlebot", "curl").
		// The path is irrelevant — the signal is the UA string, not the URL.
		if ua != "" && statsstore.IsBot(ua) {
			addReason(BotReasonUserAgent, "")
		}

		// Data-center IP — check once per IP since it's IP-level, not per-visit.
		// The path is irrelevant — the signal is the IP range, not the URL.
		if !dataCenterChecked && ip != "" && statsstore.IsDataCenterIP(ip) {
			addReason(BotReasonDataCenterIP, "")
			dataCenterChecked = true
		}

		// Referrer spam.
		// The path is irrelevant — the signal is the referrer domain, not the URL.
		if referrer != "" && statsstore.IsReferrerSpam(referrer) {
			addReason(BotReasonReferrerSpam, "")
		}

		// Bot-only path patterns (robots.txt, ads.txt, sitemap.xml, etc.).
		// Uses the same suffix-on-last-segment matching as
		// statsstore.IsBotPath to avoid false positives like
		// "/robots.txt.backup" matching "robots.txt".
		if matched := matchBotPath(path, botPatterns); matched != "" {
			addReason(matched, path)
		}

		// Universal malicious path patterns (.env, .git/, shell.php, etc.).
		// Uses the same matching as statsstore.IsMaliciousPath: suffix on
		// last segment for file patterns, segment match for directory
		// patterns (ending in "/").
		if matched := matchMaliciousPath(path, maliciousPatterns); matched != "" {
			addReason(matched, path)
		}

		// Extra consumer-specific patterns (e.g. ".php", "/wp-admin").
		// These use case-insensitive substring matching, consistent with
		// the identify-bots scan in handleIdentifyBots.
		pathLower := strings.ToLower(path)
		for i, patternLower := range extraLower {
			if strings.Contains(pathLower, patternLower) {
				addReason(extraPathPatterns[i], path)
			}
		}
	}

	list := make([]BotReasonDisplay, 0, len(reasons))
	for _, r := range reasons {
		if r.Paths == nil {
			r.Paths = []string{}
		}
		list = append(list, BotReasonDisplay{
			Pattern: r.Pattern,
			Hits:    r.Hits,
			Paths:   r.Paths,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Pattern < list[j].Pattern
	})

	return list
}

// matchBotPath finds the first bot path pattern that matches the given path
// using suffix-on-last-segment matching (same as statsstore.IsBotPath).
// Returns the matched pattern string, or "" if no match.
func matchBotPath(path string, patterns []string) string {
	if path == "" {
		return ""
	}
	pathLower := strings.ToLower(path)
	lastSeg := pathLastSegment(pathLower)
	for _, pattern := range patterns {
		if strings.HasSuffix(lastSeg, pattern) {
			return pattern
		}
	}
	return ""
}

// matchMaliciousPath finds the first malicious path pattern that matches the
// given path using the same logic as statsstore.IsMaliciousPath: suffix on
// last segment for file patterns, segment match for directory patterns
// (patterns ending in "/"). Returns the matched pattern string, or "" if
// no match.
func matchMaliciousPath(path string, patterns []string) string {
	if path == "" {
		return ""
	}
	pathLower := strings.ToLower(path)
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/") {
			dirName := strings.TrimSuffix(pattern, "/")
			for _, segment := range strings.Split(pathLower, "/") {
				if segment == dirName {
					return pattern
				}
			}
		} else {
			if strings.HasSuffix(pathLastSegment(pathLower), pattern) {
				return pattern
			}
		}
	}
	return ""
}

// pathLastSegment extracts the last path segment (filename) from a path,
// stripping query strings and fragments first. For example,
// "/courses/go" returns "go", "[GET] /robots.txt" returns "robots.txt".
// This mirrors the unexported pathLastSegment in statsstore/bot_filter.go.
func pathLastSegment(path string) string {
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}
