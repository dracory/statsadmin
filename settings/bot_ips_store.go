package settings

// This file previously contained wrapper functions for the old JSON-based bot
// IP list (getBotRecords, getBotIPs, addBotRecord, removeBotRecord) and a
// local flagIPVisitorsAsBot. These have been removed:
//   - The wrappers are dead code after the migration to visitor-record-based
//     bot tagging.
//   - flagIPVisitorsAsBot is now shared.FlagIPVisitorsAsBot in the shared
//     package, used by both settings and ipdetails controllers.
