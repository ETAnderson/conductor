package ingest

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/ETAnderson/conductor/internal/domain"
)

type UnknownKeyWarning struct {
	UnknownKeys []string `json:"unknown_keys"`
}

type ParseResult struct {
	Products []domain.Product
	Warnings UnknownKeyWarning
}

func ParseProductsAllowUnknown(body []byte) (ParseResult, error) {
	dec := json.NewDecoder(bytes.NewReader(body))

	// Expect an array of objects
	var rawItems []map[string]json.RawMessage
	if err := dec.Decode(&rawItems); err != nil {
		return ParseResult{}, err
	}

	unknown := make(map[string]struct{})

	products := make([]domain.Product, 0, len(rawItems))
	for _, item := range rawItems {
		p, itemUnknown, err := parseSingleProduct(item)
		if err != nil {
			return ParseResult{}, err
		}

		for k := range itemUnknown {
			unknown[k] = struct{}{}
		}

		products = append(products, p)
	}

	w := UnknownKeyWarning{
		UnknownKeys: setToSortedSlice(unknown),
	}

	return ParseResult{
		Products: products,
		Warnings: w,
	}, nil
}

func parseSingleProduct(item map[string]json.RawMessage) (domain.Product, map[string]struct{}, error) {
	known := knownTopLevelKeys()
	unknown := make(map[string]struct{})

	var p domain.Product

	for key := range item {
		if _, ok := known[key]; !ok {
			unknown[key] = struct{}{}
		}
	}

	// Known fields
	unmarshalIfPresent(item, "product_key", &p.ProductKey)
	unmarshalIfPresent(item, "group_key", &p.GroupKey)

	unmarshalIfPresent(item, "title", &p.Title)
	unmarshalIfPresent(item, "description", &p.Description)

	unmarshalIfPresent(item, "link", &p.Link)
	unmarshalIfPresent(item, "image_link", &p.ImageLink)
	unmarshalIfPresent(item, "additional_image_links", &p.AdditionalImageLinks)

	unmarshalIfPresent(item, "brand", &p.Brand)
	unmarshalIfPresent(item, "gtin", &p.GTIN)
	unmarshalIfPresent(item, "mpn", &p.MPN)

	unmarshalIfPresent(item, "condition", &p.Condition)
	unmarshalIfPresent(item, "availability", &p.Availability)

	unmarshalIfPresent(item, "price", &p.Price)
	unmarshalIfPresent(item, "sale_price", &p.SalePrice)

	unmarshalIfPresent(item, "options", &p.Options)
	unmarshalIfPresent(item, "attributes", &p.Attributes)

	// channel: we also collect unknown channel keys (e.g. channel.tiktok)
	if raw, ok := item["channel"]; ok {
		chUnknown, err := parseChannel(raw, &p.Channel)
		if err != nil {
			return domain.Product{}, nil, err
		}
		for k := range chUnknown {
			unknown["channel."+k] = struct{}{}
		}
	}

	// Normalize unknown keys (trim, lower?):
	// We keep original key spelling for easier customer debugging, but we do trim whitespace.
	normalized := make(map[string]struct{}, len(unknown))
	for k := range unknown {
		kk := strings.TrimSpace(k)
		if kk != "" {
			normalized[kk] = struct{}{}
		}
	}

	return p, normalized, nil
}

func parseChannel(raw json.RawMessage, out *domain.ChannelFields) (map[string]struct{}, error) {
	unknown := make(map[string]struct{})

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	known := map[string]struct{}{
		"google": {},
		"meta":   {},
		"yotpo":  {},
	}

	for k := range obj {
		if _, ok := known[k]; !ok {
			unknown[k] = struct{}{}
		}
	}

	if v, ok := obj["google"]; ok {
		var gf domain.GoogleFields
		_ = json.Unmarshal(v, &gf) // ignore unknown inside google block
		out.Google = &gf
	}
	if v, ok := obj["meta"]; ok {
		var mf domain.MetaFields
		_ = json.Unmarshal(v, &mf)
		out.Meta = &mf
	}
	if v, ok := obj["yotpo"]; ok {
		var yf domain.YotpoFields
		_ = json.Unmarshal(v, &yf)
		out.Yotpo = &yf
	}

	return unknown, nil
}

func knownTopLevelKeys() map[string]struct{} {
	return map[string]struct{}{
		"product_key":            {},
		"group_key":              {},
		"title":                  {},
		"description":            {},
		"link":                   {},
		"image_link":             {},
		"additional_image_links": {},
		"brand":                  {},
		"gtin":                   {},
		"mpn":                    {},
		"condition":              {},
		"availability":           {},
		"price":                  {},
		"sale_price":             {},
		"options":                {},
		"attributes":             {},
		"channel":                {},
	}
}

func unmarshalIfPresent[T any](obj map[string]json.RawMessage, key string, dst *T) {
	raw, ok := obj[key]
	if !ok {
		return
	}
	_ = json.Unmarshal(raw, dst) // validation catches missing/invalid required fields later
}

func setToSortedSlice(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ParseProductObjectAllowUnknown(line []byte) (domain.Product, map[string]struct{}, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(line, &obj); err != nil {
		return domain.Product{}, nil, err
	}

	return parseSingleProduct(obj)
}

func SortedUnknownKeys(set map[string]struct{}) []string {
	return setToSortedSlice(set)
}
