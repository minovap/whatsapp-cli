package api

import "strings"

// PhoneFilter enforces phone number whitelist/blacklist rules on JIDs.
// Matching uses the last 6 digits of the phone portion (before the @ sign).
type PhoneFilter struct {
	whitelist []string
	blacklist []string
}

// NewPhoneFilter creates a PhoneFilter from config whitelist/blacklist entries.
func NewPhoneFilter(whitelist, blacklist []string) *PhoneFilter {
	return &PhoneFilter{
		whitelist: whitelist,
		blacklist: blacklist,
	}
}

// IsAllowed returns true if the JID passes the filter rules.
// Group JIDs (@g.us) always pass.
// If whitelist is non-empty, only matching JIDs are allowed (blacklist ignored).
// If only blacklist is set, matching JIDs are blocked.
// If neither is set, all JIDs are allowed.
func (f *PhoneFilter) IsAllowed(jid string) bool {
	// Group JIDs always pass
	if strings.HasSuffix(jid, "@g.us") {
		return true
	}

	suffix := extractSuffix(jid)

	if len(f.whitelist) > 0 {
		return matchesAny(suffix, f.whitelist)
	}

	if len(f.blacklist) > 0 {
		return !matchesAny(suffix, f.blacklist)
	}

	return true
}

// extractSuffix returns the last 6 digits of the phone portion of a JID.
// For "1234567890@s.whatsapp.net", it returns "567890".
func extractSuffix(jid string) string {
	phone := jid
	if at := strings.Index(jid, "@"); at >= 0 {
		phone = jid[:at]
	}

	if len(phone) > 6 {
		return phone[len(phone)-6:]
	}
	return phone
}

// matchesAny checks if the suffix matches any entry in the list.
// Each entry is also reduced to its last 6 digits for comparison.
func matchesAny(suffix string, entries []string) bool {
	for _, entry := range entries {
		entrySuffix := entry
		if len(entry) > 6 {
			entrySuffix = entry[len(entry)-6:]
		}
		if suffix == entrySuffix {
			return true
		}
	}
	return false
}

// JIDSuffixes returns the last-6-digit suffixes formatted for SQL LIKE patterns.
// Each entry becomes "<last6digits>@%" so the store layer can use LIKE '%567890@%'.
func (f *PhoneFilter) JIDSuffixes() (includeJIDs, excludeJIDs []string) {
	for _, entry := range f.whitelist {
		suffix := entry
		if len(entry) > 6 {
			suffix = entry[len(entry)-6:]
		}
		includeJIDs = append(includeJIDs, suffix+"@")
	}
	for _, entry := range f.blacklist {
		suffix := entry
		if len(entry) > 6 {
			suffix = entry[len(entry)-6:]
		}
		excludeJIDs = append(excludeJIDs, suffix+"@")
	}
	return
}
