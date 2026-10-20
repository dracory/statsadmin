package shared

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/dracory/statsstore"
)

// SettingKeyBotIPs is the key under which bot IP records are stored as a JSON
// array in the stats store's settings table.
const SettingKeyBotIPs = "bot_ips"

// BotReason captures a single piece of evidence that caused an IP to be
// flagged as a bot (e.g. it visited sitemap.xml 3 times).
type BotReason struct {
	Pattern string `json:"pattern"` // the matched bot-page pattern (e.g. "sitemap.xml")
	Path    string `json:"path"`    // sample visitor paths
	Hits    int    `json:"hits"`    // how many times this pattern was hit
}

// BotRecord is the full record for a single flagged bot IP.
type BotRecord struct {
	IP      string      `json:"ip"`
	AddedAt string      `json:"added_at"` // "YYYY-MM-DD HH:MM:SS" UTC
	Source  string      `json:"source"`   // "identified" or "manual"
	Reasons []BotReason `json:"reasons"`
}

// BotSourceIdentified means the IP was added via the "Identify Bots" scan.
const BotSourceIdentified = "identified"

// BotSourceManual means the IP was added manually by the user.
const BotSourceManual = "manual"

// GetBotRecords reads the full bot IP records from the stats store settings.
func GetBotRecords(ctx context.Context, store statsstore.StoreInterface) []BotRecord {
	if store == nil {
		return []BotRecord{}
	}

	raw, err := store.SettingGet(ctx, SettingKeyBotIPs)
	if err != nil || raw == "" {
		return []BotRecord{}
	}

	var records []BotRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return []BotRecord{}
	}
	return records
}

// GetBotIPs returns just the IP strings from the bot records.
func GetBotIPs(ctx context.Context, store statsstore.StoreInterface) []string {
	records := GetBotRecords(ctx, store)
	ips := make([]string, 0, len(records))
	for _, r := range records {
		ips = append(ips, r.IP)
	}
	return ips
}

// SaveBotRecords writes the full bot records to the stats store settings.
func SaveBotRecords(ctx context.Context, store statsstore.StoreInterface, records []BotRecord) error {
	if store == nil {
		return errors.New("stats store not available")
	}

	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	return store.SettingSet(ctx, SettingKeyBotIPs, string(raw))
}

// AddBotRecord adds a single bot record, avoiding duplicates by IP.
func AddBotRecord(ctx context.Context, store statsstore.StoreInterface, record BotRecord) error {
	records := GetBotRecords(ctx, store)
	for _, r := range records {
		if r.IP == record.IP {
			return errors.New("IP already in bot list")
		}
	}
	if record.AddedAt == "" {
		record.AddedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	records = append(records, record)
	return SaveBotRecords(ctx, store, records)
}

// AddBotRecords batch-adds multiple bot records in a single store write,
// skipping any IPs that already exist in the list.
func AddBotRecords(ctx context.Context, store statsstore.StoreInterface, newRecords []BotRecord) error {
	existing := GetBotRecords(ctx, store)
	existingSet := map[string]bool{}
	for _, r := range existing {
		existingSet[r.IP] = true
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	for _, rec := range newRecords {
		if existingSet[rec.IP] {
			continue
		}
		if rec.AddedAt == "" {
			rec.AddedAt = now
		}
		existing = append(existing, rec)
		existingSet[rec.IP] = true
	}
	return SaveBotRecords(ctx, store, existing)
}

// RemoveBotRecord removes a bot record by IP.
func RemoveBotRecord(ctx context.Context, store statsstore.StoreInterface, ip string) error {
	records := GetBotRecords(ctx, store)
	found := false
	filtered := make([]BotRecord, 0, len(records))
	for _, r := range records {
		if r.IP == ip {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return errors.New("IP not found in bot list")
	}
	return SaveBotRecords(ctx, store, filtered)
}
