package layout

// cacheSetter toggles component-level caching.
type cacheSetter interface {
	SetCacheEnabled(enabled bool)
}

// cacheGetter reports component-level caching state.
type cacheGetter interface {
	CacheEnabled() bool
}

// SetCacheEnabled enables or disables caching for a component when supported.
// Children still follow their own caching rules.
func SetCacheEnabled(c Component, enabled bool) {
	if setter, ok := c.(cacheSetter); ok && setter != nil {
		setter.SetCacheEnabled(enabled)
	}
}

// CacheEnabled reports whether a component caches its own surface (default true).
func CacheEnabled(c Component) bool {
	if getter, ok := c.(cacheGetter); ok && getter != nil {
		return getter.CacheEnabled()
	}
	return true
}
