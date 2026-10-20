package shared

const (
	CONTROLLER_DASHBOARD  = "dashboard"
	CONTROLLER_VISITORS   = "visitors"
	CONTROLLER_SESSIONS   = "sessions"
	CONTROLLER_SETTINGS   = "settings"
	CONTROLLER_IP_DETAILS = "ip-details"
)

// Query parameter names
const (
	ParamController = "controller"
	ParamAction     = "action"
	ParamPeriod     = "period"
	ParamCountry    = "country"
	ParamDeviceType = "device_type"
	ParamPath       = "path"
	ParamDateFrom   = "date_from"
	ParamDateTo     = "date_to"
	ParamPage       = "page"
	ParamIP         = "ip"
)

// Period identifiers
const (
	PeriodToday     = "today"
	PeriodYesterday = "yesterday"
	PeriodLast7Days = "last-7-days"
	PeriodThisMonth = "this-month"
	PeriodLastMonth = "last-month"
	PeriodAllTime   = "all-time"
	PeriodDefault   = PeriodLast7Days
)

// Pagination
const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)
