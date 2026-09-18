package dto

import (
	"encoding/json"
	"strings"
)

// ContactEntry is one admin-configured customer-service contact entry.
//
// 魔改 #12: 后台可以自由新增/删除联系客服条目。每条自带图标、名称、类型、
// 展示方式（弹窗 / 悬停 / 内联）与打开方式（新标签页 / 当前标签页），
// 因此同一个站点可以同时展示 Telegram 群、微信群、二维码和纯文本。
//
// Keep this shape in sync with service/contact_entries.go (legacyContactEntry):
// the service package must not import handler/dto (dto imports service), so the
// JSON tags are redeclared there.
type ContactEntry struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	Label       string `json:"label"`
	IconType    string `json:"icon_type"` // "emoji" | "image"
	Icon        string `json:"icon"`
	Type        string `json:"type"` // "link" | "qrcode" | "text"
	URL         string `json:"url,omitempty"`
	QRCode      string `json:"qr_code,omitempty"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Display     string `json:"display"`     // "modal" | "hover" | "inline"
	OpenTarget  string `json:"open_target"` // "new_tab" | "current_tab"
	SortOrder   int    `json:"sort_order"`
}

// ParseContactEntries parses the #12 contact-entry JSON array.
// Empty or invalid input yields an empty (never nil) slice.
func ParseContactEntries(raw string) []ContactEntry {
	items := decodeContactEntries(raw)
	if items == nil {
		return []ContactEntry{}
	}
	return items
}

// ParseEnabledContactEntries returns only the entries that are enabled.
// Used by the public settings endpoint so a disabled entry never leaks out.
func ParseEnabledContactEntries(raw string) []ContactEntry {
	items := decodeContactEntries(raw)
	out := make([]ContactEntry, 0, len(items))
	for _, item := range items {
		if item.Enabled {
			out = append(out, item)
		}
	}
	return out
}

func decodeContactEntries(raw string) []ContactEntry {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var items []ContactEntry
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}
