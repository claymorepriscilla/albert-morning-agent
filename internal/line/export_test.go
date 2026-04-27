package line

// SetPushURL allows tests to override the LINE push endpoint.
func SetPushURL(url string) { pushURL = url }

// SetBroadcastURL allows tests to override the LINE broadcast endpoint.
func SetBroadcastURL(url string) { broadcastURL = url }
