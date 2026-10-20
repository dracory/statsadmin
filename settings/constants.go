package settings

const (
	actionLoadIPs        = "load-ips"
	actionAddIP          = "add-ip"
	actionRemoveIP       = "remove-ip"
	actionDeleteVisitors = "delete-visitors"

	// Bot IP actions
	actionLoadBots      = "load-bots"
	actionAddBot        = "add-bot"
	actionRemoveBot     = "remove-bot"
	actionIdentifyBots  = "identify-bots"
	actionDeleteBots    = "delete-bots"
	actionDeleteThreats = "delete-threats"
	actionLoadStats     = "load-stats"

	FieldIPs          = "ips"
	FieldIP           = "ip"
	FieldTotal        = "total"
	FieldDeletedCount = "deleted_count"
	FieldTotalRecords = "total_records"
	FieldOlderCount   = "older_count"

	// Bot IP fields
	FieldBots    = "bots"
	FieldBotIPs  = "bot_ips"
	FieldBotIP   = "bot_ip"
	FieldBotList = "bot_list"

	// Identify-bots response fields
	FieldCandidates     = "candidates"
	FieldCandidateIP    = "candidate_ip"
	FieldCandidateHits  = "candidate_hits"
	FieldCandidatePaths = "candidate_paths"
	FieldCandidateUAs   = "candidate_user_agents"
	FieldAlreadyListed  = "already_listed"
	FieldReasons        = "reasons"
	FieldAutoAdded      = "auto_added"
	FieldAutoAddedCount = "auto_added_count"

	// Bot record fields (for load-bots response)
	FieldHits     = "hits"
	FieldLastSeen = "last_seen"

	// Consumer-specific bot file patterns (not in statsstore's universal
	// list because they are too broad for general use).
	botPagePatternGoogleVerify = "google"
	// /.well-known/ is RFC 8615's reserved URI prefix for site-wide
	// metadata (assetlinks.json, apple-app-site-association, ACME challenges,
	// OIDC discovery, etc.). Only automated clients — crawlers, domain
	// verifiers, app-link checkers — ever request it; no human browser
	// navigates there. Any hit is non-human.
	botPagePatternWellKnown = "/.well-known/"

	// Stack-dependent malicious patterns — this is a Go site, so requests
	// for these file types are scanners probing for non-existent attack
	// surface. Universal malicious patterns (.env, .git/, .svn/, .htpasswd,
	// shell.php) come from statsstore.MaliciousPathPatterns().
	botPagePatternPhp     = ".php"
	botPagePatternPy      = ".py"
	botPagePatternAsp     = ".asp"
	botPagePatternAspx    = ".aspx"
	botPagePatternJsp     = ".jsp"
	botPagePatternCgi     = ".cgi"
	botPagePatternPl      = ".pl"
	botPagePatternRb      = ".rb"
	botPagePatternWp          = "/wp"
	botPagePatternWordPress   = "/wordpress"
	botPagePatternWpAdmin     = "/wp-admin"
	botPagePatternWpLogin     = "/wp-login"
	botPagePatternWpIncludes  = "/wp-includes"
	botPagePatternWpContent   = "/wp-content"
	botPagePatternWlwManifest = "wlwmanifest.xml"
	botPagePatternXmlRpc      = "/xmlrpc.php"
	botPagePatternConfig      = "/config.php"
	botPagePatternAdminer     = "/adminer.php"
)
