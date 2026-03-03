package redis

import "fmt"

// Key-builder helpers centralise every Redis key pattern in one place.
// All keys are prefixed with "mf:" to namespace MasterFabric keys on a
// shared Redis instance.

// RefreshTokenKey returns the Redis key used to store a refresh token.
// Pattern: mf:refresh:<userID>:<token>
func RefreshTokenKey(userID, token string) string {
	return fmt.Sprintf("mf:refresh:%s:%s", userID, token)
}

// TokenBlacklistKey returns the Redis key used to blacklist an access token.
// Pattern: mf:blacklist:<token>
func TokenBlacklistKey(token string) string {
	return fmt.Sprintf("mf:blacklist:%s", token)
}

// UserSettingsCacheKey returns the Redis key used to cache user settings.
// Pattern: mf:settings:user:<userID>
func UserSettingsCacheKey(userID string) string {
	return fmt.Sprintf("mf:settings:user:%s", userID)
}

// AppSettingsCacheKey returns the Redis key used to cache an app-settings
// entry by its string key.
// Pattern: mf:settings:app:<key>
func AppSettingsCacheKey(key string) string {
	return fmt.Sprintf("mf:settings:app:%s", key)
}
