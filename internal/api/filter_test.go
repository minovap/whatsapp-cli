package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhoneFilter_NeitherSet(t *testing.T) {
	f := NewPhoneFilter(nil, nil)

	assert.True(t, f.IsAllowed("1234567890@s.whatsapp.net"))
	assert.True(t, f.IsAllowed("9876543210@s.whatsapp.net"))
	assert.True(t, f.IsAllowed("120363123456789012@g.us"))
}

func TestPhoneFilter_WhitelistOnly(t *testing.T) {
	f := NewPhoneFilter([]string{"1234567890"}, nil)

	// Matches last 6 digits "567890"
	assert.True(t, f.IsAllowed("1234567890@s.whatsapp.net"))
	// Different country code, same last 6 digits
	assert.True(t, f.IsAllowed("44234567890@s.whatsapp.net"))
	// Different last 6 digits — blocked
	assert.False(t, f.IsAllowed("9876543210@s.whatsapp.net"))
}

func TestPhoneFilter_BlacklistOnly(t *testing.T) {
	f := NewPhoneFilter(nil, []string{"9876543210"})

	// Not in blacklist — allowed
	assert.True(t, f.IsAllowed("1234567890@s.whatsapp.net"))
	// In blacklist (matches last 6 digits "543210")
	assert.False(t, f.IsAllowed("9876543210@s.whatsapp.net"))
	// Different country code, same last 6 digits — also blocked
	assert.False(t, f.IsAllowed("44876543210@s.whatsapp.net"))
}

func TestPhoneFilter_BothSet_WhitelistWins(t *testing.T) {
	// When both are set, whitelist takes precedence, blacklist is ignored
	f := NewPhoneFilter([]string{"1234567890"}, []string{"1234567890"})

	// Matches whitelist — allowed despite also matching blacklist
	assert.True(t, f.IsAllowed("1234567890@s.whatsapp.net"))
	// Not in whitelist — blocked
	assert.False(t, f.IsAllowed("9876543210@s.whatsapp.net"))
}

func TestPhoneFilter_GroupJIDs_AlwaysPass(t *testing.T) {
	// Even with restrictive whitelist, group JIDs pass
	f := NewPhoneFilter([]string{"1234567890"}, nil)

	assert.True(t, f.IsAllowed("120363123456789012@g.us"))
	assert.True(t, f.IsAllowed("999999999999@g.us"))

	// Same with blacklist
	f2 := NewPhoneFilter(nil, []string{"1234567890"})
	assert.True(t, f2.IsAllowed("120363123456789012@g.us"))
}

func TestPhoneFilter_ShortPhoneNumbers(t *testing.T) {
	// Phone number shorter than 6 digits
	f := NewPhoneFilter([]string{"12345"}, nil)

	// Exact match on short number
	assert.True(t, f.IsAllowed("12345@s.whatsapp.net"))
	// Longer number whose last 6 are not "12345"
	assert.False(t, f.IsAllowed("9912345@s.whatsapp.net"))
}

func TestPhoneFilter_MultipleEntries(t *testing.T) {
	f := NewPhoneFilter([]string{"1234567890", "1112223333"}, nil)

	// First entry matches
	assert.True(t, f.IsAllowed("1234567890@s.whatsapp.net"))
	// Second entry matches (last 6 = "223333")
	assert.True(t, f.IsAllowed("1112223333@s.whatsapp.net"))
	// Neither matches
	assert.False(t, f.IsAllowed("9999999999@s.whatsapp.net"))
}

func TestPhoneFilter_NoAtSign(t *testing.T) {
	f := NewPhoneFilter([]string{"1234567890"}, nil)

	// JID without @ sign — still extracts last 6 digits
	assert.True(t, f.IsAllowed("1234567890"))
	assert.False(t, f.IsAllowed("9876543210"))
}

func TestPhoneFilter_EmptySlices(t *testing.T) {
	// Empty slices (not nil) should behave same as nil
	f := NewPhoneFilter([]string{}, []string{})

	assert.True(t, f.IsAllowed("1234567890@s.whatsapp.net"))
	assert.True(t, f.IsAllowed("9876543210@s.whatsapp.net"))
}
