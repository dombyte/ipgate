package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dombyte/ipgate/internal/ipmatcher"
	"github.com/dombyte/ipgate/internal/models"
)

// ClearCacheHandler clears the entire cache
func ClearCacheHandler(deps *models.HandlerDeps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.Cache == nil {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{"status": "cache_not_enabled"}); err != nil {
				// Log error but continue
			}
			return
		}
		deps.Cache.Clear()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "cache_cleared"}); err != nil {
			// Log error but continue
		}
	})
}

// CacheDumpHandler returns cache statistics and contents
func CacheDumpHandler(deps *models.HandlerDeps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.Cache == nil {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "cache_not_enabled",
				"message": "Cache is disabled in configuration",
			}); err != nil {
				// Log error but continue
			}
			return
		}
		stats := deps.Cache.GetStats()
		entries := deps.Cache.GetEntries()

		// Convert entries to a format that can be JSON encoded
		entryList := make([]map[string]interface{}, 0, len(entries))
		for ip, entry := range entries {
			entryList = append(entryList, map[string]interface{}{
				"ip":        ip,
				"status":    entry.Status,
				"timestamp": entry.Timestamp.Unix(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"stats":   stats,
			"entries": entryList,
		}); err != nil {
			// Log error but continue
		}
	})
}

// ConfigHandler returns the current configuration with metadata and IP lists
func ConfigHandler(deps *models.HandlerDeps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := deps.Config

		// Get IP list information from matcher if available
		var whitelistCount, blocklistCount int
		var whitelistEntries, blocklistEntries []string

		if config.IPMatcher != nil {
			if matcher, ok := config.IPMatcher.(*ipmatcher.IPMatcher); ok {
				whitelistCount = matcher.GetWhitelistSize()
				blocklistCount = matcher.GetBlocklistSize()

				// Get actual entries if debug tracking is enabled
				if config.DebugEndpoint {
					whitelistEntries = matcher.GetWhitelistEntries()
					blocklistEntries = matcher.GetBlocklistEntries()
				}
			}
		}

		// Create response with organized structure
		response := map[string]interface{}{
			"server": map[string]interface{}{
				"port":         config.Port,
				"error_page":   config.ErrorPage,
				"error_format": config.ErrorFormat,
			},
			"status_codes": map[string]interface{}{
				"allowed": config.StatusAllowed,
				"denied":  config.StatusDenied,
			},
			"cache": map[string]interface{}{
				"enabled":              config.Cache.Enabled,
				"ttl":                  config.Cache.TTL,
				"max_entries":          config.Cache.MaxEntries,
				"prune_interval":       config.Cache.PruneInterval,
				"shard_count":          config.Cache.ShardCount,
				"prune_on_get":         config.Cache.PruneOnGet,
				"write_buffer_size":    config.Cache.WriteBufferSize,
				"auto_clear_on_change": config.Cache.AutoClearOnChange,
			},
			"features": map[string]interface{}{
				"debug_endpoint":      config.DebugEndpoint,
				"watch_files_enabled": config.WatchFilesEnabled,
			},
			"limits": map[string]interface{}{
				"blocklist_max_size": config.BlocklistMaxSize,
			},
			"files": map[string]interface{}{
				"whitelist": config.WhitelistFiles,
				"blacklist": config.BlacklistFiles,
			},
			"remotes": map[string]interface{}{
				"whitelist": config.WhitelistRemotes,
				"blacklist": config.BlacklistRemotes,
			},
			"ip_lists": map[string]interface{}{
				"whitelist_count": whitelistCount,
				"blocklist_count": blocklistCount,
				"whitelist":       whitelistEntries,
				"blocklist":       blocklistEntries,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log error but continue
		}
	})
}
