package replicated

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// licenseField mirrors the per-field object returned by the SDK's
// GET /api/v1/license/fields endpoint.
// Value is any because the SDK encodes it as the native JSON type
// (boolean, number, or string) rather than always quoting it as a string.
type licenseField struct {
	Name      string `json:"name"`
	Title     string `json:"title"`
	Value     any    `json:"value"`
	ValueType string `json:"valueType"`
}

// LicenseClient reads custom license field values from the Replicated
// in-cluster SDK API. Results are cached for five minutes.
type LicenseClient struct {
	sdkURL   string
	client   *http.Client
	mu       sync.Mutex
	cached   map[string]licenseField
	cachedAt time.Time
}

func NewLicenseClient(sdkURL string) *LicenseClient {
	return &LicenseClient{
		sdkURL: sdkURL,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *LicenseClient) fields(ctx context.Context) (map[string]licenseField, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.cachedAt.IsZero() && time.Since(c.cachedAt) < 5*time.Minute {
		return c.cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sdkURL+"/api/v1/license/fields", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sdk request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sdk response: status %d", resp.StatusCode)
	}

	var fields map[string]licenseField
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	c.cached = fields
	c.cachedAt = time.Now()
	return fields, nil
}

// AllowUserCreation reports whether the license permits creating new users.
// Defaults to true when the field is absent or the SDK is unreachable, so
// that development environments (no SDK) are unaffected.
func (c *LicenseClient) AllowUserCreation(ctx context.Context) (bool, error) {
	fields, err := c.fields(ctx)
	if err != nil {
		return true, err
	}
	f, ok := fields["allow_user_creation"]
	if !ok {
		return true, nil
	}
	switch v := f.Value.(type) {
	case bool:
		return v, nil
	case string:
		return v == "true", nil
	default:
		return fmt.Sprintf("%v", f.Value) == "true", nil
	}
}
