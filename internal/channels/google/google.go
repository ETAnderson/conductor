package google

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ETAnderson/conductor/internal/channels"
	"github.com/ETAnderson/conductor/internal/state"
)

type Channel struct {
	Store state.Store
}

func (c Channel) Name() string { return "google" }

type money struct {
	AmountDecimal string `json:"amount_decimal"`
	Currency      string `json:"currency"`
}

type normalizedProduct struct {
	ProductKey   string `json:"product_key"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Link         string `json:"link"`
	ImageLink    string `json:"image_link"`
	Condition    string `json:"condition"`
	Availability string `json:"availability"`
	Price        money  `json:"price"`
}

func (c Channel) Build(ctx context.Context, tenantID uint64, products []channels.ProductRef) (channels.BuildResult, error) {
	if c.Store == nil {
		return channels.BuildResult{}, fmt.Errorf("google channel: store is nil")
	}

	out := channels.BuildResult{
		Channel: c.Name(),
		Attempt: 1,
		Items:   make([]channels.ProductOutcome, 0, len(products)),
	}

	for _, ref := range products {
		doc, ok, err := c.Store.GetProductDoc(ctx, tenantID, ref.ProductKey)
		if err != nil {
			out.ErrCount++
			out.Items = append(out.Items, channels.ProductOutcome{
				ProductKey: ref.ProductKey,
				Status:     "error",
				Message:    "read_product_doc_failed",
			})
			continue
		}
		if !ok {
			out.ErrCount++
			out.Items = append(out.Items, channels.ProductOutcome{
				ProductKey: ref.ProductKey,
				Status:     "error",
				Message:    "missing_product_doc",
			})
			continue
		}

		var p normalizedProduct
		if err := json.Unmarshal(doc.ProductJSON, &p); err != nil {
			out.ErrCount++
			out.Items = append(out.Items, channels.ProductOutcome{
				ProductKey: ref.ProductKey,
				Status:     "error",
				Message:    "invalid_product_json",
			})
			continue
		}

		// v0 validation: ensure required Google-ish baseline
		if strings.TrimSpace(p.Description) == "" {
			out.ErrCount++
			out.Items = append(out.Items, channels.ProductOutcome{
				ProductKey: ref.ProductKey,
				Status:     "error",
				Message:    "missing_description",
			})
			continue
		}

		if strings.TrimSpace(p.Title) == "" ||
			strings.TrimSpace(p.Link) == "" ||
			strings.TrimSpace(p.ImageLink) == "" ||
			strings.TrimSpace(p.Condition) == "" ||
			strings.TrimSpace(p.Availability) == "" ||
			strings.TrimSpace(p.Price.AmountDecimal) == "" ||
			strings.TrimSpace(p.Price.Currency) == "" {
			out.ErrCount++
			out.Items = append(out.Items, channels.ProductOutcome{
				ProductKey: ref.ProductKey,
				Status:     "error",
				Message:    "missing_required_fields",
			})
			continue
		}

		_ = GoogleItem{
			ID:           p.ProductKey,
			Title:        p.Title,
			Description:  p.Description,
			Link:         p.Link,
			ImageLink:    p.ImageLink,
			Availability: p.Availability,
			Condition:    p.Condition,
			Price:        fmt.Sprintf("%s %s", p.Price.AmountDecimal, p.Price.Currency),
		}

		out.OkCount++
		out.Items = append(out.Items, channels.ProductOutcome{
			ProductKey: ref.ProductKey,
			Status:     "ok",
			Message:    "google_item_built",
		})
	}

	return out, nil
}
