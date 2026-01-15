package redemption

// Product defines a financial product's core parameters
type Product struct {
	ID           string
	Name         string
	APY          float64
	DurationDays int
	AutoRenew    bool
}

// Built-in product catalog (can extend with more products later)
var products = map[string]Product{
	"prod-7d": {
		ID:           "prod-7d",
		Name:         "7天固定收益",
		APY:          0.07,
		DurationDays: 7,
		AutoRenew:    true,
	},
}

// GetProduct returns a product by its ID
func GetProduct(productID string) (*Product, bool) {
	p, ok := products[productID]
	if !ok {
		return nil, false
	}
	return &p, true
}
