// Package store defines the key-value store interface and implementations
package store

// KeyValuePair represents a key-value pair for bulk operations (legacy)
// Deprecated: Use Entry type instead
type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// StoreStats represents statistics about the store (legacy)
// Deprecated: Use Metrics type instead
type StoreStats struct {
	TotalKeys      int `json:"total_keys"`
	TotalRequests  int `json:"total_requests"`
	GetRequests    int `json:"get_requests"`
	SetRequests    int `json:"set_requests"`
	DeleteRequests int `json:"delete_requests"`
}
