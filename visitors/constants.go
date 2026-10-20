package visitors

const (
	actionLoadVisitors = "load-visitors"
	actionExportCSV    = "export-csv"

	FieldVisitors    = "visitors"
	FieldTotal       = "total"
	FieldPage        = "page"
	FieldPerPage     = "per_page"
	FieldTotalPages  = "total_pages"
	FieldConditions  = "conditions"
	FieldID          = "id"
	FieldIP          = "ip"
	FieldCountry     = "country"
	FieldCountryName = "country_name"
	FieldPath        = "path"
	FieldBrowser     = "browser"
	FieldOS          = "os"
	FieldDevice      = "device"
	FieldDeviceType  = "device_type"
	FieldCreatedAt   = "created_at"
	FieldUserAgent   = "user_agent"
	FieldIsBot       = "is_bot"
	FieldIsThreat    = "is_threat"

	// IP details action
	actionLoadIPDetails = "load-ip-details"

	// IP details response fields
	FieldDetails     = "details"
	FieldFirstSeen   = "first_seen"
	FieldLastSeen    = "last_seen"
	FieldVisitCount  = "visit_count"
	FieldBrowserVer  = "browser_version"
	FieldOSVer       = "os_version"
	FieldAcceptLang  = "accept_language"
	FieldAcceptEnc   = "accept_encoding"
	FieldReferrer    = "referrer"
	FieldFingerprint = "fingerprint"
	FieldPaths       = "paths"
	FieldUniquePaths = "unique_paths"

	// Condition fields (what can be filtered on)
	CondFieldIP         = "ip"
	CondFieldCountry    = "country"
	CondFieldDeviceType = "device_type"
	CondFieldPathCont   = "path_contains"
	CondFieldPathExact  = "path_exact"
	CondFieldBrowser    = "browser"
	CondFieldOS         = "os"
	CondFieldDateFrom   = "date_from"
	CondFieldDateTo     = "date_to"
	CondFieldIsBot      = "is_bot"
	CondFieldIsThreat   = "is_threat"

	// Condition operators
	CondOpEquals   = "equals"
	CondOpContains = "contains"

	// Is-bot filter values
	CondValueBotYes = "yes"
	CondValueBotNo  = "no"

	// JSON keys for condition objects
	CondKeyField    = "field"
	CondKeyOperator = "operator"
	CondKeyValue    = "value"
)

// ConditionOption defines a selectable filter field in the UI.
type ConditionOption struct {
	Value       string
	Label       string
	Operators   []string
	InputType   string // "text", "date", "select"
	Placeholder string
}

// ConditionOptions returns the ordered list of filterable fields for the UI.
func ConditionOptions() []ConditionOption {
	return []ConditionOption{
		{Value: CondFieldIP, Label: "IP Address", Operators: []string{CondOpEquals}, InputType: "text", Placeholder: "e.g. 192.168.1.1"},
		{Value: CondFieldCountry, Label: "Country", Operators: []string{CondOpEquals}, InputType: "text", Placeholder: "ISO2 code (e.g. GB)"},
		{Value: CondFieldDeviceType, Label: "Device Type", Operators: []string{CondOpEquals}, InputType: "select", Placeholder: ""},
		{Value: CondFieldPathCont, Label: "Path contains", Operators: []string{CondOpContains}, InputType: "text", Placeholder: "e.g. /admin"},
		{Value: CondFieldPathExact, Label: "Path equals", Operators: []string{CondOpEquals}, InputType: "text", Placeholder: "e.g. [GET] /"},
		{Value: CondFieldBrowser, Label: "Browser", Operators: []string{CondOpEquals}, InputType: "text", Placeholder: "e.g. Chrome"},
		{Value: CondFieldOS, Label: "Operating System", Operators: []string{CondOpEquals}, InputType: "text", Placeholder: "e.g. Windows"},
		{Value: CondFieldDateFrom, Label: "Created after", Operators: []string{CondOpEquals}, InputType: "date", Placeholder: ""},
		{Value: CondFieldDateTo, Label: "Created before", Operators: []string{CondOpEquals}, InputType: "date", Placeholder: ""},
		{Value: CondFieldIsBot, Label: "Is Bot", Operators: []string{CondOpEquals}, InputType: "select", Placeholder: ""},
		{Value: CondFieldIsThreat, Label: "Is Threat", Operators: []string{CondOpEquals}, InputType: "select", Placeholder: ""},
	}
}

// DeviceTypeOptions lists the selectable device types.
func DeviceTypeOptions() []string {
	return []string{"desktop", "mobile", "tablet", "bot"}
}
