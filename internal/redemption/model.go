package redemption

import "time"

type RedemptionStatus string

const (
	StatusHolding  RedemptionStatus = "HOLDING"
	StatusRedeemed RedemptionStatus = "REDEEMED"
	StatusRenewed  RedemptionStatus = "RENEWED"
	StatusFailed   RedemptionStatus = "FAILED"
)

type RedemptionRecord struct {
	ID               string           `json:"id"`
	UserID           string           `json:"userId"`
	ProductID        string           `json:"productId"`
	Principal        float64          `json:"principal"`
	APY              float64          `json:"apy"`
	DurationDays     int              `json:"durationDays"`
	Status           RedemptionStatus `json:"status"`
	StartDate        time.Time        `json:"startDate"`
	EndDate          time.Time        `json:"endDate"`
	AutoRenew        bool             `json:"autoRenew"`
	InterestTotal    float64          `json:"interestTotal"`
	RedemptionAmount float64          `json:"redemptionAmount"`
	RedeemedAt       *time.Time       `json:"redeemedAt,omitempty"`
	RenewedToOrderID *string          `json:"renewedToOrderId,omitempty"`
}
