package gemini

// SetEndpoint allows tests to override the Groq API endpoint.
// Compiled only during testing.
func SetEndpoint(url string) { groqEndpoint = url }
