package admin

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	maxContactEntries  = 20
	maxContactLabelLen = 50
	maxContactURLLen   = 2048
	maxContactValueLen = 200
	maxContactDescLen  = 200
	maxContactGroupLen = 30
	maxContactIconLen  = 500 * 1024
	maxContactQRLen    = 2 * 1024 * 1024
	maxContactIDLen    = 32
)

// validateContactEntries validates and serialises the #12 contact-entry list.
//
// A nil request field means "keep the stored value" (the field was omitted),
// while an explicit empty array clears the list — the same semantics the
// custom-menu-items editor uses.
func validateContactEntries(req *[]dto.ContactEntry, previous string, c *gin.Context) (string, bool) {
	if req == nil {
		return previous, true
	}

	entries := *req
	if len(entries) > maxContactEntries {
		response.BadRequest(c, "Too many contact entries (max 20)")
		return "", false
	}

	for i := range entries {
		item := &entries[i]

		item.Label = strings.TrimSpace(item.Label)
		if item.Label == "" {
			response.BadRequest(c, "Contact entry label is required")
			return "", false
		}
		if len([]rune(item.Label)) > maxContactLabelLen {
			response.BadRequest(c, "Contact entry label is too long (max 50 characters)")
			return "", false
		}

		item.IconType = strings.TrimSpace(item.IconType)
		if item.IconType == "" {
			item.IconType = "emoji"
		}
		if item.IconType != "emoji" && item.IconType != "image" {
			response.BadRequest(c, "Contact entry icon_type must be 'emoji' or 'image'")
			return "", false
		}
		item.Icon = strings.TrimSpace(item.Icon)
		if item.IconType == "image" {
			if item.Icon == "" {
				response.BadRequest(c, "Contact entry image icon is required when icon_type is 'image'")
				return "", false
			}
			if len(item.Icon) > maxContactIconLen {
				response.BadRequest(c, "Contact entry image icon is too large (max 500KB)")
				return "", false
			}
			if !strings.HasPrefix(item.Icon, "data:image/") {
				if err := config.ValidateAbsoluteHTTPURL(item.Icon); err != nil {
					response.BadRequest(c, "Contact entry image icon must be a data:image/* or an absolute http(s) URL")
					return "", false
				}
			}
		}

		item.Type = strings.TrimSpace(item.Type)
		if item.Type == "" {
			item.Type = "link"
		}
		switch item.Type {
		case "link":
			item.URL = strings.TrimSpace(item.URL)
			if item.URL == "" {
				response.BadRequest(c, "Contact entry URL is required for link entries")
				return "", false
			}
			if len(item.URL) > maxContactURLLen {
				response.BadRequest(c, "Contact entry URL is too long (max 2048 characters)")
				return "", false
			}
			if err := config.ValidateAbsoluteHTTPURL(item.URL); err != nil {
				response.BadRequest(c, "Contact entry URL must be an absolute http(s) URL")
				return "", false
			}
		case "qrcode":
			item.QRCode = strings.TrimSpace(item.QRCode)
			if item.QRCode == "" {
				response.BadRequest(c, "Contact entry QR code image is required for qrcode entries")
				return "", false
			}
			if len(item.QRCode) > maxContactQRLen {
				response.BadRequest(c, "Contact entry QR code image is too large (max 2MB)")
				return "", false
			}
			if !strings.HasPrefix(item.QRCode, "data:image/") {
				if err := config.ValidateAbsoluteHTTPURL(item.QRCode); err != nil {
					response.BadRequest(c, "Contact entry QR code must be a data:image/* or an absolute http(s) URL")
					return "", false
				}
			}
		case "text":
			item.Value = strings.TrimSpace(item.Value)
			if item.Value == "" {
				response.BadRequest(c, "Contact entry text value is required for text entries")
				return "", false
			}
			if len([]rune(item.Value)) > maxContactValueLen {
				response.BadRequest(c, "Contact entry text value is too long (max 200 characters)")
				return "", false
			}
		default:
			response.BadRequest(c, "Contact entry type must be 'link', 'qrcode' or 'text'")
			return "", false
		}

		if len([]rune(item.Description)) > maxContactDescLen {
			response.BadRequest(c, "Contact entry description is too long (max 200 characters)")
			return "", false
		}

		// 魔改 #23: 可选分组名。留空表示不分组。
		item.Group = strings.TrimSpace(item.Group)
		if len([]rune(item.Group)) > maxContactGroupLen {
			response.BadRequest(c, "Contact entry group is too long (max 30 characters)")
			return "", false
		}

		item.Display = strings.TrimSpace(item.Display)
		if item.Display == "" {
			item.Display = "modal"
		}
		if item.Display != "modal" && item.Display != "hover" && item.Display != "inline" {
			response.BadRequest(c, "Contact entry display must be 'modal', 'hover' or 'inline'")
			return "", false
		}

		item.OpenTarget = strings.TrimSpace(item.OpenTarget)
		if item.OpenTarget == "" {
			item.OpenTarget = "new_tab"
		}
		if item.OpenTarget != "new_tab" && item.OpenTarget != "current_tab" {
			response.BadRequest(c, "Contact entry open_target must be 'new_tab' or 'current_tab'")
			return "", false
		}

		item.ID = strings.TrimSpace(item.ID)
		if item.ID == "" {
			id, err := generateMenuItemID()
			if err != nil {
				response.Error(c, 500, "Failed to generate contact entry ID")
				return "", false
			}
			item.ID = id
		} else {
			if len(item.ID) > maxContactIDLen {
				response.BadRequest(c, "Contact entry ID is too long (max 32 characters)")
				return "", false
			}
			if !menuItemIDPattern.MatchString(item.ID) {
				response.BadRequest(c, "Contact entry ID may only contain a-z, A-Z, 0-9, - and _")
				return "", false
			}
		}

		item.SortOrder = i
	}

	seen := make(map[string]struct{}, len(entries))
	for _, item := range entries {
		if _, dup := seen[item.ID]; dup {
			response.BadRequest(c, "Duplicate contact entry ID: "+item.ID)
			return "", false
		}
		seen[item.ID] = struct{}{}
	}

	blob, err := json.Marshal(entries)
	if err != nil {
		response.BadRequest(c, "Failed to serialize contact entries")
		return "", false
	}
	return string(blob), true
}
