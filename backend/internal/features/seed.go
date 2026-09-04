package features

// Seeds defines the default feature flags inserted on first run.
// Add new flags here before implementing the feature itself.
var Seeds = []Flag{
	{Name: "live_streaming", Enabled: true, Description: "Live broadcast feature"},
	{Name: "track_uploads", Enabled: true, Description: "Broadcaster track audio upload via presigned S3 URLs"},
	{Name: "chat_websocket", Enabled: false, Description: "Live chat between listeners of the same stream"},
	{Name: "recommendations", Enabled: false, Description: "Listen-history-based track recommendations"},
	{Name: "offline_mode", Enabled: false, Description: "Playlist caching for offline playback"},
	{Name: "transcoding", Enabled: false, Description: "Adaptive bitrate transcoding based on client bandwidth"},
	{Name: "playlists", Enabled: true, Description: "Playlist creation and management"},
	{Name: "favorites", Enabled: true, Description: "Favoriting tracks, streams, and playlists"},
	{Name: "admin_supervision", Enabled: true, Description: "Admin supervision dashboard, incident feed and alert notifications"},
	{Name: "account_deletion", Enabled: true, Description: "Self-service account deletion via DELETE /users/me"},
}
