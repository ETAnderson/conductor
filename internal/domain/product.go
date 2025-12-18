package domain

type Money struct {
	AmountDecimal string `json:"amount_decimal"`
	Currency      string `json:"currency"`
}

type Product struct {
	ProductKey string `json:"product_key"`
	GroupKey   string `json:"group_key,omitempty"`

	Title       string `json:"title"`
	Description string `json:"description"`

	Link      string `json:"link"`
	ImageLink string `json:"image_link"`

	AdditionalImageLinks []string `json:"additional_image_links,omitempty"`

	Brand string `json:"brand,omitempty"`
	GTIN  string `json:"gtin,omitempty"`
	MPN   string `json:"mpn,omitempty"`

	Condition    string `json:"condition"`
	Availability string `json:"availability"`

	Price     Money  `json:"price"`
	SalePrice *Money `json:"sale_price,omitempty"`

	Options    map[string]string `json:"options,omitempty"`
	Attributes map[string]any    `json:"attributes,omitempty"`

	Channel ChannelFields `json:"channel"`
}

type ChannelFields struct {
	Google *GoogleFields `json:"google,omitempty"`
	Meta   *MetaFields   `json:"meta,omitempty"`
	Yotpo  *YotpoFields  `json:"yotpo,omitempty"`
}
