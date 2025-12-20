package google

type GoogleItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Link         string `json:"link"`
	ImageLink    string `json:"image_link"`
	Availability string `json:"availability"`
	Condition    string `json:"condition"`
	Price        string `json:"price"` // "19.99 USD"
}
