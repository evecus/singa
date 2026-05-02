package profile

import (
	"encoding/json"
	"time"
)

// Profile is a named wizard configuration that is independent of any
// subscription. It can optionally reference a subscription by ID so that
// nodes are injected at start time, but creating / deleting a subscription
// does NOT affect profiles.
type Profile struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	SubscriptionID string          `json:"subscriptionId,omitempty"` // optional link
	UpdatedAt      time.Time       `json:"updatedAt"`
	WizardConfig   json.RawMessage `json:"wizardConfig,omitempty"`
}
