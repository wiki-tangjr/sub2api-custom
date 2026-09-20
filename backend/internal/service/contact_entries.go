package service

import (
	"encoding/json"
	"strings"
)

// legacyContactEntry mirrors dto.ContactEntry. The service package must not
// import handler/dto (dto imports service), so the shape is redeclared here.
// Keep the JSON tags in sync with dto.ContactEntry.
type legacyContactEntry struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Label       string `json:"label"`
	IconType    string `json:"icon_type"`
	Icon        string `json:"icon"`
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	QRCode      string `json:"qr_code,omitempty"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Group       string `json:"group,omitempty"`
	Display     string `json:"display"`
	OpenTarget  string `json:"open_target"`
	SortOrder   int    `json:"sort_order"`
}

// resolveContactEntries returns the effective #12 contact-entry JSON array.
//
// Backward compatibility: when the new `contact_entries` setting is absent or
// empty (a site that has not been migrated yet), the legacy flat settings
// (telegram_group_url / wechat_group_qr_code / contact_info) are converted into
// an equivalent entry list so the existing production config keeps rendering.
func resolveContactEntries(settings map[string]string) string {
	raw := strings.TrimSpace(settings[SettingKeyContactEntries])
	if raw != "" && raw != "[]" && json.Valid([]byte(raw)) {
		return raw
	}

	entries := make([]legacyContactEntry, 0, 3)

	if u := strings.TrimSpace(settings[SettingKeyTelegramGroupURL]); u != "" {
		label := strings.TrimSpace(settings[SettingKeyTelegramEntryLabel])
		if label == "" {
			label = "加入 Telegram 群组"
		}
		entries = append(entries, legacyContactEntry{
			ID: "legacy-telegram", Enabled: true, Label: label,
			IconType: "emoji", Icon: "\u2708\ufe0f", Type: "link", URL: u,
			Display: "modal", OpenTarget: "new_tab", SortOrder: len(entries),
		})
	}

	if qr := strings.TrimSpace(settings[SettingKeyWeChatGroupQRCode]); qr != "" {
		label := strings.TrimSpace(settings[SettingKeyWeChatGroupEntryLabel])
		if label == "" {
			label = "扫码加入微信群"
		}
		entries = append(entries, legacyContactEntry{
			ID: "legacy-wechat-group", Enabled: true, Label: label,
			IconType: "emoji", Icon: "\U0001f4ac", Type: "qrcode", QRCode: qr,
			Display: "modal", OpenTarget: "new_tab", SortOrder: len(entries),
		})
	}

	if info := strings.TrimSpace(settings[SettingKeyContactInfo]); info != "" {
		label := strings.TrimSpace(settings[SettingKeyWeChatContactEntryLabel])
		if label == "" {
			label = "添加微信客服"
		}
		entries = append(entries, legacyContactEntry{
			ID: "legacy-wechat-contact", Enabled: true, Label: label,
			IconType: "emoji", Icon: "\U0001f4ac", Type: "text", Value: info,
			Display: "hover", OpenTarget: "new_tab", SortOrder: len(entries),
		})
	}

	blob, err := json.Marshal(entries)
	if err != nil {
		return "[]"
	}
	return string(blob)
}

// resolvePublicContactEntries is resolveContactEntries plus an enabled-only
// filter, so a disabled entry (and its QR-code payload) never reaches an
// unauthenticated visitor through either /api/v1/settings/public or SSR.
func resolvePublicContactEntries(settings map[string]string) string {
	raw := resolveContactEntries(settings)
	items, err := decodeContactEntriesJSON(raw)
	if err != nil {
		return "[]"
	}
	out := make([]legacyContactEntry, 0, len(items))
	for _, item := range items {
		if item.Enabled {
			out = append(out, item)
		}
	}
	blob, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(blob)
}

func decodeContactEntriesJSON(raw string) ([]legacyContactEntry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var items []legacyContactEntry
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// normalizeContactEntriesJSON guards the persisted #12 blob so a malformed
// value can never reach the settings table.
func normalizeContactEntriesJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || !json.Valid([]byte(raw)) {
		return "[]"
	}
	return raw
}
