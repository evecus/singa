package subscription

import (
	"encoding/json"
	"time"
)

// Subscription represents an imported proxy subscription URL plus
// the full wizard config the user configured for this subscription.
type Subscription struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	URL          string          `json:"url"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	NodeCount    int             `json:"nodeCount"`
	Error        string          `json:"error,omitempty"`
	WizardConfig json.RawMessage `json:"wizardConfig,omitempty"` // full wizard form JSON
}

// ProxyNode is a single outbound entry in sing-box JSON format stored in the cache.
type ProxyNode struct {
	Tag string         `json:"tag"`
	Type string        `json:"type"`
	Raw map[string]any `json:"-"` // full sing-box outbound object
}
