package gold

// SetAPIURL allows tests to override the gold price API endpoint.
// Compiled only during testing.
func SetAPIURL(url string) { apiURL = url }
