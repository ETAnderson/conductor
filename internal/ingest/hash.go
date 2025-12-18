package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/ETAnderson/conductor/internal/domain"
)

type Hasher struct{}

func (h Hasher) HashNormalized(p domain.Product) (string, error) {
	n := normalizeForHash(p)

	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// normalizeForHash builds a deterministic representation of the product for hashing.
// - sorts map keys
// - sorts additional image links
// - preserves only canonical fields (no DB/run metadata)
func normalizeForHash(p domain.Product) any {
	// Copy and sort additional images (treat order as irrelevant)
	additional := make([]string, len(p.AdditionalImageLinks))
	copy(additional, p.AdditionalImageLinks)
	sort.Strings(additional)

	// Sort options map
	options := sortedStringMap(p.Options)

	// Sort attributes map (string keys, any values). Values must be JSON-marshalable.
	attrs := sortedAnyMap(p.Attributes)

	// Channel blocks: include only control.state for now (v1).
	// This ensures lifecycle state changes trigger deltas.
	ch := map[string]any{}

	if p.Channel.Google != nil {
		ch["google"] = map[string]any{
			"control": map[string]any{"state": p.Channel.Google.Control.State},
		}
	}
	if p.Channel.Meta != nil {
		ch["meta"] = map[string]any{
			"control": map[string]any{"state": p.Channel.Meta.Control.State},
		}
	}
	if p.Channel.Yotpo != nil {
		ch["yotpo"] = map[string]any{
			"control": map[string]any{"state": p.Channel.Yotpo.Control.State},
		}
	}

	// Canonical envelope
	return map[string]any{
		"product_key": p.ProductKey,
		"group_key":   p.GroupKey,

		"title":       p.Title,
		"description": p.Description,

		"link":       p.Link,
		"image_link": p.ImageLink,

		"additional_image_links": additional,

		"brand": p.Brand,
		"gtin":  p.GTIN,
		"mpn":   p.MPN,

		"condition":    p.Condition,
		"availability": p.Availability,

		"price": map[string]any{
			"amount_decimal": p.Price.AmountDecimal,
			"currency":       p.Price.Currency,
		},
		"sale_price": salePriceMap(p.SalePrice),

		"options":    options,
		"attributes": attrs,

		"channel": ch,
	}
}

func salePriceMap(m *domain.Money) any {
	if m == nil {
		return nil
	}
	return map[string]any{
		"amount_decimal": m.AmountDecimal,
		"currency":       m.Currency,
	}
}

func sortedStringMap(m map[string]string) []any {
	if len(m) == 0 {
		return []any{}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{
			"k": k,
			"v": m[k],
		})
	}

	return out
}

func sortedAnyMap(m map[string]any) []any {
	if len(m) == 0 {
		return []any{}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{
			"k": k,
			"v": m[k],
		})
	}

	return out
}
