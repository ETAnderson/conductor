package ingest

import (
	"fmt"
	"strings"

	"github.com/ETAnderson/conductor/internal/domain"
)

type ValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationResult struct {
	Issues []ValidationIssue `json:"issues"`
}

func (r ValidationResult) IsValid() bool {
	return len(r.Issues) == 0
}

func ValidateProductBase(p domain.Product) ValidationResult {
	var res ValidationResult

	// Required base fields (v1)
	requireNonEmpty(&res, "product_key", p.ProductKey)
	requireNonEmpty(&res, "title", p.Title)
	requireNonEmpty(&res, "description", p.Description)
	requireNonEmpty(&res, "link", p.Link)
	requireNonEmpty(&res, "image_link", p.ImageLink)
	requireNonEmpty(&res, "condition", p.Condition)
	requireNonEmpty(&res, "availability", p.Availability)
	requireNonEmpty(&res, "price.amount_decimal", p.Price.AmountDecimal)
	requireNonEmpty(&res, "price.currency", p.Price.Currency)

	// Basic sanity checks (cheap)
	if p.Price.AmountDecimal != "" && !looksLikeDecimal(p.Price.AmountDecimal) {
		addIssue(&res, "price.amount_decimal", "invalid_decimal", "amount_decimal must look like a decimal number (e.g. \"19.99\")")
	}
	if p.Price.Currency != "" && len(p.Price.Currency) != 3 {
		addIssue(&res, "price.currency", "invalid_currency", "currency must be a 3-letter ISO code (e.g. \"USD\")")
	}

	return res
}

func ValidateChannelControls(p domain.Product, enabledChannels []string) ValidationResult {
	var res ValidationResult

	enabled := make(map[string]struct{}, len(enabledChannels))
	for _, c := range enabledChannels {
		enabled[strings.ToLower(strings.TrimSpace(c))] = struct{}{}
	}

	// For each enabled channel: require presence and validate control.state
	for ch := range enabled {
		switch ch {
		case "google":
			if p.Channel.Google == nil {
				addIssue(&res, "channel.google", "missing_channel_block", "google channel block is required because google is enabled for this feed")
				continue
			}
			validateControlState(&res, "channel.google.control.state", p.Channel.Google.Control.State)
		case "meta":
			if p.Channel.Meta == nil {
				addIssue(&res, "channel.meta", "missing_channel_block", "meta channel block is required because meta is enabled for this feed")
				continue
			}
			validateControlState(&res, "channel.meta.control.state", p.Channel.Meta.Control.State)
		case "yotpo":
			if p.Channel.Yotpo == nil {
				addIssue(&res, "channel.yotpo", "missing_channel_block", "yotpo channel block is required because yotpo is enabled for this feed")
				continue
			}
			validateControlState(&res, "channel.yotpo.control.state", p.Channel.Yotpo.Control.State)
		default:
			// Unknown channel enabled at feed level.
			// We don't fail the product, but we do record a warning-like issue.
			addIssue(&res, fmt.Sprintf("channel.%s", ch), "unknown_channel", "channel is enabled but not recognized by this service version")
		}
	}

	return res
}

func validateControlState(res *ValidationResult, path string, state domain.ChannelLifecycleState) {
	switch state {
	case domain.ChannelStateActive, domain.ChannelStateInactive, domain.ChannelStateDelete:
		return
	default:
		addIssue(res, path, "invalid_state", "state must be one of: active, inactive, delete")
	}
}

func requireNonEmpty(res *ValidationResult, path string, v string) {
	if strings.TrimSpace(v) == "" {
		addIssue(res, path, "required", "field is required")
	}
}

func addIssue(res *ValidationResult, path string, code string, msg string) {
	res.Issues = append(res.Issues, ValidationIssue{
		Path:    path,
		Code:    code,
		Message: msg,
	})
}

func looksLikeDecimal(v string) bool {
	// Cheap check: digits with optional single dot.
	dot := 0
	digit := 0

	for _, r := range v {
		if r == '.' {
			dot++
			if dot > 1 {
				return false
			}
			continue
		}
		if r >= '0' && r <= '9' {
			digit++
			continue
		}
		return false
	}

	return digit > 0
}
