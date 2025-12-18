package ingest

import (
	"testing"

	"github.com/ETAnderson/conductor/internal/domain"
)

func validBaseProduct() domain.Product {
	return domain.Product{
		ProductKey:   "sku1",
		Title:        "Test",
		Description:  "Desc",
		Link:         "https://example.com/p/sku1",
		ImageLink:    "https://example.com/p/sku1.jpg",
		Condition:    "new",
		Availability: "in_stock",
		Price: domain.Money{
			AmountDecimal: "19.99",
			Currency:      "USD",
		},
		Channel: domain.ChannelFields{
			Google: &domain.GoogleFields{
				Control: domain.ChannelControl{State: domain.ChannelStateActive},
			},
		},
	}
}

func TestValidateProductBase_RequiredFields(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(p *domain.Product)
		wantIssueKey string
	}{
		{"missing product_key", func(p *domain.Product) { p.ProductKey = "" }, "product_key"},
		{"missing title", func(p *domain.Product) { p.Title = "" }, "title"},
		{"missing description", func(p *domain.Product) { p.Description = "" }, "description"},
		{"missing link", func(p *domain.Product) { p.Link = "" }, "link"},
		{"missing image_link", func(p *domain.Product) { p.ImageLink = "" }, "image_link"},
		{"missing condition", func(p *domain.Product) { p.Condition = "" }, "condition"},
		{"missing availability", func(p *domain.Product) { p.Availability = "" }, "availability"},
		{"missing price.amount_decimal", func(p *domain.Product) { p.Price.AmountDecimal = "" }, "price.amount_decimal"},
		{"missing price.currency", func(p *domain.Product) { p.Price.Currency = "" }, "price.currency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validBaseProduct()
			tt.mutate(&p)

			res := ValidateProductBase(p)
			if res.IsValid() {
				t.Fatalf("expected invalid result")
			}

			if !hasIssuePath(res, tt.wantIssueKey) {
				t.Fatalf("expected issue for path %q, got %#v", tt.wantIssueKey, res.Issues)
			}
		})
	}
}

func TestValidateProductBase_DecimalAndCurrency(t *testing.T) {
	t.Run("invalid decimal", func(t *testing.T) {
		p := validBaseProduct()
		p.Price.AmountDecimal = "19.9.9"

		res := ValidateProductBase(p)
		if res.IsValid() {
			t.Fatalf("expected invalid result")
		}
		if !hasIssueCode(res, "invalid_decimal") {
			t.Fatalf("expected invalid_decimal, got %#v", res.Issues)
		}
	})

	t.Run("invalid currency length", func(t *testing.T) {
		p := validBaseProduct()
		p.Price.Currency = "US"

		res := ValidateProductBase(p)
		if res.IsValid() {
			t.Fatalf("expected invalid result")
		}
		if !hasIssueCode(res, "invalid_currency") {
			t.Fatalf("expected invalid_currency, got %#v", res.Issues)
		}
	})
}

func TestValidateChannelControls_RequiredChannelBlock(t *testing.T) {
	p := validBaseProduct()
	p.Channel.Google = nil

	res := ValidateChannelControls(p, []string{"google"})
	if res.IsValid() {
		t.Fatalf("expected invalid result")
	}
	if !hasIssueCode(res, "missing_channel_block") {
		t.Fatalf("expected missing_channel_block, got %#v", res.Issues)
	}
}

func TestValidateChannelControls_InvalidState(t *testing.T) {
	p := validBaseProduct()
	p.Channel.Google.Control.State = "bogus"

	res := ValidateChannelControls(p, []string{"google"})
	if res.IsValid() {
		t.Fatalf("expected invalid result")
	}
	if !hasIssueCode(res, "invalid_state") {
		t.Fatalf("expected invalid_state, got %#v", res.Issues)
	}
}

func TestValidateChannelControls_UnknownEnabledChannel(t *testing.T) {
	p := validBaseProduct()

	res := ValidateChannelControls(p, []string{"notarealchannel"})
	if res.IsValid() {
		t.Fatalf("expected invalid/issue result (unknown channel should emit issue)")
	}
	if !hasIssueCode(res, "unknown_channel") {
		t.Fatalf("expected unknown_channel, got %#v", res.Issues)
	}
}

func hasIssuePath(res ValidationResult, path string) bool {
	for _, it := range res.Issues {
		if it.Path == path {
			return true
		}
	}
	return false
}

func hasIssueCode(res ValidationResult, code string) bool {
	for _, it := range res.Issues {
		if it.Code == code {
			return true
		}
	}
	return false
}
