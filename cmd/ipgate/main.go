package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dombyte/ipgate/internal/blocklist"
	"github.com/dombyte/ipgate/internal/cache"
	"github.com/dombyte/ipgate/internal/config"
	"github.com/dombyte/ipgate/internal/ipmatcher"
	"github.com/dombyte/ipgate/internal/logging"
	"github.com/dombyte/ipgate/internal/models"
	"github.com/dombyte/ipgate/internal/routes"
	"github.com/robfig/cron/v3"
)

var logger *logging.Logger

func main() {
	// Parse CLI flags
	configFile, logLevel := parseCLIFlags()

	// Initialize logger
	logger = logging.NewLogger(logLevel)
	logger.Info("Starting IPGate application")
	logger.Debug("Log level set to", "level", logLevel)

	// Load configuration
	cfg, err := loadConfiguration(configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", "error", err)
	}

	// Setup IP matcher
	if err := setupIPMatcher(cfg); err != nil {
		logger.Fatal("Failed to setup IP matcher: %v", "error", err)
	}

	// Initialize cache
	cacheInstance, err := initializeCache(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize cache: %v", "error", err)
	}

	// Setup cron scheduler
	setupCronScheduler(cfg, cacheInstance)

	// Setup file watchers
	setupFileWatchers(cfg, cacheInstance)

	// Create handler dependencies and start server
	startServer(cfg, cacheInstance)
}

// loadConfiguration loads and parses the application configuration
func loadConfiguration(configFile string) (*config.Config, error) {
	// Load configuration using new API
	configAPI := config.NewConfigAPI()
	if err := configAPI.LoadConfig(configFile); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Get fully populated config
	cfg, err := configAPI.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Debug: Dump configuration
	logger.Debug("=== CONFIGURATION DUMP ===")
	logger.Debug("Port", "port", cfg.Port)
	logger.Debug("Debug Endpoint", "debug_endpoint", cfg.DebugEndpoint)
	logger.Debug("Watch Files Enabled", "watch_files_enabled", cfg.WatchFilesEnabled)
	logger.Debug("Error Page", "error_page", cfg.ErrorPage)
	logger.Debug("Error Format", "error_format", cfg.ErrorFormat)
	logger.Debug("Status Allowed", "status_allowed", cfg.StatusAllowed)
	logger.Debug("Status Denied", "status_denied", cfg.StatusDenied)
	logger.Debug("Blocklist Max Size", "blocklist_max_size", cfg.BlocklistMaxSize)
	logger.Debug("Headers", "headers", cfg.Headers)
	logger.Debug("Cache Enabled", "cache_enabled", cfg.Cache.Enabled)
	logger.Debug("Cache TTL", "cache_ttl", cfg.Cache.TTL)
	logger.Debug("Cache Max Entries", "cache_max_entries", cfg.Cache.MaxEntries)
	logger.Debug("Cache Prune Interval", "cache_prune_interval", cfg.Cache.PruneInterval)
	logger.Debug("Cache Shard Count", "cache_shard_count", cfg.Cache.ShardCount)
	logger.Debug("Cache Prune On Get", "cache_prune_on_get", cfg.Cache.PruneOnGet)
	logger.Debug("Cache Write Buffer Size", "cache_write_buffer_size", cfg.Cache.WriteBufferSize)
	logger.Debug("Cache Auto Clear On Change", "cache_auto_clear_on_change", cfg.Cache.AutoClearOnChange)

	// Debug: Dump file configurations
	logger.Debug("=== WHITELIST FILES ===")
	for i, file := range cfg.WhitelistFiles {
		logger.Debug("Whitelist File", "index", i, "path", file.Path)
	}
	logger.Debug("=== WHITELIST REMOTES ===")
	for i, remote := range cfg.WhitelistRemotes {
		logger.Debug("Whitelist Remote", "index", i, "url", remote.URL, "cron", remote.Cron)
	}
	logger.Debug("=== BLACKLIST FILES ===")
	for i, file := range cfg.BlacklistFiles {
		logger.Debug("Blacklist File", "index", i, "path", file.Path)
	}
	logger.Debug("=== BLACKLIST REMOTES ===")
	for i, remote := range cfg.BlacklistRemotes {
		logger.Debug("Blacklist Remote", "index", i, "url", remote.URL, "cron", remote.Cron)
	}
	logger.Debug("=== END CONFIGURATION DUMP ===")

	return cfg, nil
}

// setupIPMatcher loads blocklists, whitelists, and creates IP matcher
func setupIPMatcher(cfg *config.Config) error {
	// Load blocklists
	logger.Debug("Loading blocklists (include remotes: true)")
	loadBlocklists(cfg, true)
	logger.Debug("Loaded blocklists", "count", len(cfg.Blocklists))
	for i, blocklist := range cfg.Blocklists {
		logger.Debug("Blocklist", "index", i, "ips", len(blocklist))
	}

	// Load whitelists
	logger.Debug("Loading whitelists (include remotes: true)")
	loadWhitelists(cfg, true)
	logger.Debug("Loaded whitelist", "count", len(cfg.Whitelist))

	// Load IPMatcher for the application
	logger.Debug("Loading IPMatcher")
	matcher := loadIPMatcher(cfg)
	if matcher == nil {
		return fmt.Errorf("failed to load IPMatcher")
	}
	cfg.IPMatcher = matcher
	logger.Debug("Successfully loaded IPMatcher", "whitelist", len(cfg.Whitelist), "blocklist", len(cfg.Blocklists))

	return nil
}

// initializeCache creates and starts the cache instance
func initializeCache(cfg *config.Config) (*cache.Cache, error) {
	// Initialize cache (optional)
	var cacheInstance *cache.Cache
	if cfg.Cache.Enabled {
		cacheInstance = cache.NewCache(
			cfg.Cache.TTL,
			cfg.Cache.MaxEntries,
			cfg.Cache.PruneInterval,
			cfg.Cache.ShardCount,
			cfg.Cache.PruneOnGet,
			cfg.Cache.WriteBufferSize,
		)
		cacheInstance.StartPruner()
		cache.SetCacheInstance(cacheInstance) // Set the package-level cache instance
	}

	return cacheInstance, nil
}

// setupCronScheduler sets up scheduled updates for remote blocklists and whitelists
func setupCronScheduler(cfg *config.Config, cacheInstance *cache.Cache) {
	startCronScheduler(cfg, logger, cacheInstance)
}

// setupFileWatchers sets up file watchers for local blocklists and whitelists
func setupFileWatchers(cfg *config.Config, cacheInstance *cache.Cache) {
	// Start file watcher for local blacklists and whitelists
	if cfg.WatchFilesEnabled {
		// Extract file paths
		blacklistPaths := extractFilePaths(cfg.BlacklistFiles)
		whitelistPaths := extractFilePaths(cfg.WhitelistFiles)

		// Setup blacklist watcher
		if len(blacklistPaths) > 0 {
			setupBlacklistWatcher(cfg, cacheInstance, blacklistPaths)
		}

		// Setup whitelist watcher
		if len(whitelistPaths) > 0 {
			setupWhitelistWatcher(cfg, cacheInstance, whitelistPaths)
		}
	}
}

// extractFilePaths extracts file paths from file configurations
func extractFilePaths(files interface{}) []string {
	switch files := files.(type) {
	case []config.WhitelistFile:
		var paths []string
		for _, file := range files {
			paths = append(paths, file.Path)
		}
		return paths
	case []config.LocalFile:
		var paths []string
		for _, file := range files {
			paths = append(paths, file.Path)
		}
		return paths
	default:
		return []string{}
	}
}

// setupBlacklistWatcher sets up file watcher for blacklist files
func setupBlacklistWatcher(cfg *config.Config, cacheInstance *cache.Cache, paths []string) {
	if err := blocklist.WatchBlocklists(paths, func() {
		logger.Info("Local blacklists changed, reloading...")
		loadBlocklists(cfg, true) // Reload both local and remote files
		newMatcher := loadIPMatcher(cfg)
		if newMatcher != nil {
			cfg.IPMatcher = newMatcher
		}
		handleCacheClearOnChange(cfg, cacheInstance, "blacklist")
	}, logger); err != nil {
		logger.Error("Failed to setup blacklist watcher: %v", "error", err)
	}
}

// setupWhitelistWatcher sets up file watcher for whitelist files
func setupWhitelistWatcher(cfg *config.Config, cacheInstance *cache.Cache, paths []string) {
	if err := blocklist.WatchBlocklists(paths, func() {
		logger.Info("Whitelist files changed, reloading...")
		loadWhitelists(cfg, true) // Reload both local and remote files
		newMatcher := loadIPMatcher(cfg)
		if newMatcher != nil {
			cfg.IPMatcher = newMatcher
		}
		handleCacheClearOnChange(cfg, cacheInstance, "whitelist")
	}, logger); err != nil {
		logger.Error("Failed to setup whitelist watcher: %v", "error", err)
	}
}

// handleCacheClearOnChange handles cache clearing when files change
func handleCacheClearOnChange(cfg *config.Config, cacheInstance *cache.Cache, fileType string) {
	if cfg.Cache.AutoClearOnChange && cacheInstance != nil {
		logger.Info("Auto-clearing cache due to " + fileType + " changes")
		cacheInstance.Clear() // Clear cache when files change
	} else {
		logger.Info(fileType + " changed but auto-clear is disabled. Cache will expire naturally based on TTL.")
	}
}

// startServer creates handler dependencies and starts the HTTP server
func startServer(cfg *config.Config, cacheInstance *cache.Cache) {
	// Create handler dependencies
	handlerDeps := &models.HandlerDeps{
		Config: cfg,
		Cache:  cacheInstance,
		Logger: logger,
	}

	// Setup router using the routes package (includes all endpoints including debug)
	router := routes.NewRouter(handlerDeps)

	// Start HTTP server
	logger.Info("Starting HTTP server on port", "port", cfg.Port)
	err := listenAndServe(cfg.Port, router)
	if err != nil {
		logger.Fatal("HTTP server failed: %v", "error", err)
	}
}

func loadBlocklists(cfg *config.Config, reloadRemote bool) {
	loadBlocklistsWithLogger(cfg, reloadRemote, logger)
}

func loadBlocklistsWithLogger(cfg *config.Config, reloadRemote bool, log *logging.Logger) {
	var blocklists [][]string

	// Load local blacklists
	blocklists = loadLocalBlocklists(cfg, log, blocklists)

	// Load remote blacklists only if requested
	if reloadRemote {
		blocklists = loadRemoteBlocklists(cfg, log, blocklists)
	}

	// Store the loaded blocklists
	cfg.Blocklists = blocklists
}

// loadLocalBlocklists loads blocklist entries from local files
func loadLocalBlocklists(cfg *config.Config, log *logging.Logger, blocklists [][]string) [][]string {
	if log != nil {
		log.Debug("Processing blacklist files", "count", len(cfg.BlacklistFiles))
	}
	for i, file := range cfg.BlacklistFiles {
		if log != nil {
			log.Debug("Processing blacklist file", "index", i, "path", file.Path)
		}
		entries, err := blocklist.LoadLocalFile(file.Path, cfg.BlocklistMaxSize)
		if err != nil {
			if log != nil {
				log.Warn("Failed to load Blocklist file", "path", file.Path, "error", err)
			}
			continue
		}
		blocklists = append(blocklists, entries)
		if log != nil {
			log.Info("Successfully loaded Blocklist file", "path", file.Path, "entries", len(entries))
		}
	}
	return blocklists
}

// loadRemoteBlocklists loads blocklist entries from remote URLs
func loadRemoteBlocklists(cfg *config.Config, log *logging.Logger, blocklists [][]string) [][]string {
	for _, remote := range cfg.BlacklistRemotes {
		entries, err := blocklist.LoadRemoteFile(remote.URL, cfg.BlocklistMaxSize)
		if err != nil {
			if log != nil {
				log.Warn("Failed to load remote Blocklist", "url", remote.URL, "error", err)
			}
			continue
		}
		blocklists = append(blocklists, entries)
		if log != nil {
			log.Info("Successfully loaded remote Blocklist", "url", remote.URL, "entries", len(entries))
		}
	}
	return blocklists
}

func loadWhitelists(cfg *config.Config, reloadRemote bool) {
	loadWhitelistsWithLogger(cfg, reloadRemote, logger)
}

func loadWhitelistsWithLogger(cfg *config.Config, reloadRemote bool, log *logging.Logger) {
	var whitelist []string

	// Load local whitelists
	whitelist = loadLocalWhitelists(cfg, log, whitelist)

	// Load remote whitelists only if requested
	if reloadRemote {
		whitelist = loadRemoteWhitelists(cfg, log, whitelist)
	}

	// Store the loaded whitelist
	cfg.Whitelist = whitelist
}

// loadLocalWhitelists loads whitelist entries from local files
func loadLocalWhitelists(cfg *config.Config, log *logging.Logger, whitelist []string) []string {
	if log != nil {
		log.Debug("Processing whitelist files", "count", len(cfg.WhitelistFiles))
	}
	for i, file := range cfg.WhitelistFiles {
		if log != nil {
			log.Debug("Processing whitelist file", "index", i, "path", file.Path)
		}
		entries, err := blocklist.LoadLocalFile(file.Path, cfg.BlocklistMaxSize)
		if err != nil {
			if log != nil {
				log.Warn("Failed to load whitelist file", "path", file.Path, "error", err)
			}
			continue
		}
		whitelist = append(whitelist, entries...)
		if log != nil {
			log.Info("Successfully loaded whitelist file", "path", file.Path, "entries", len(entries))
		}
	}
	return whitelist
}

// loadRemoteWhitelists loads whitelist entries from remote URLs
func loadRemoteWhitelists(cfg *config.Config, log *logging.Logger, whitelist []string) []string {
	for _, remote := range cfg.WhitelistRemotes {
		entries, err := blocklist.LoadRemoteFile(remote.URL, cfg.BlocklistMaxSize)
		if err != nil {
			if log != nil {
				log.Warn("Failed to load remote whitelist", "url", remote.URL, "error", err)
			}
			continue
		}
		whitelist = append(whitelist, entries...)
		if log != nil {
			log.Info("Successfully loaded remote whitelist", "url", remote.URL, "entries", len(entries))
		}
	}
	return whitelist
}

// loadIPMatcher creates an IPMatcher instance for the application
func loadIPMatcher(cfg *config.Config) *ipmatcher.IPMatcher {
	// Create new IPMatcher instance
	matcher := ipmatcher.NewIPMatcher()

	// Enable debug tracking if debug endpoint is enabled in config
	if cfg.DebugEndpoint {
		matcher.EnableDebugTracking(true)
	}

	// Load whitelist
	if err := matcher.LoadWhitelist(cfg.Whitelist); err != nil {
		logger.Warn("Failed to load whitelist: %v", "error", err)
		return nil
	}

	// Load blocklist - flatten all blocklist slices into one
	var allBlocklistEntries []string
	for _, blocklist := range cfg.Blocklists {
		allBlocklistEntries = append(allBlocklistEntries, blocklist...)
	}

	if err := matcher.LoadBlocklist(allBlocklistEntries); err != nil {
		logger.Warn("Failed to load blocklist: %v", "error", err)
		return nil
	}

	logger.Info("Successfully loaded IPMatcher",
		"whitelist", matcher.GetWhitelistSize(), "blocklist", matcher.GetBlocklistSize())

	return matcher
}
// old listenAndServe func gets flaged with gosec G114
// func listenAndServe(port string, handler http.Handler) error {
// 	return http.ListenAndServe(":"+port, handler)

// }
func listenAndServe(port string, handler http.Handler) error {
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
		// Set timeouts to avoid Slowloris attacks and resource exhaustion
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
	return srv.ListenAndServe()
}


// startCronScheduler creates and starts a cron scheduler for automatic updates
// of remote blocklists and whitelists based on the cron expressions in the configuration.
func startCronScheduler(cfg *config.Config, logger *logging.Logger, cacheInstance *cache.Cache) {
	cronScheduler := cron.New()

	// Setup remote blacklist updates
	setupBlacklistJobs(cfg, cronScheduler, logger, cacheInstance)

	// Setup remote whitelist updates
	setupWhitelistJobs(cfg, cronScheduler, logger, cacheInstance)

	// Start the scheduler if needed
	startSchedulerIfNeeded(cronScheduler, logger, cacheInstance)
}

// setupBlacklistJobs sets up cron jobs for remote blacklist updates
func setupBlacklistJobs(cfg *config.Config, cronScheduler *cron.Cron, logger *logging.Logger, cacheInstance *cache.Cache) {
	for _, remote := range cfg.BlacklistRemotes {
		if remote.Cron != "" && remote.URL != "" {
			logger.Debug("Adding cron job for blacklist", "url", remote.URL, "cron", remote.Cron)
			_, err := cronScheduler.AddFunc(remote.Cron, createBlacklistUpdateJob(cfg, logger, remote.URL, cacheInstance))
			if err != nil {
				logger.Error("Failed to add blacklist cron job: %v", "error", err)
			}
			logger.Debug("After adding blacklist job, entries count", "count", len(cronScheduler.Entries()))
		}
	}
}

// setupWhitelistJobs sets up cron jobs for remote whitelist updates
func setupWhitelistJobs(cfg *config.Config, cronScheduler *cron.Cron, logger *logging.Logger, cacheInstance *cache.Cache) {
	for _, remote := range cfg.WhitelistRemotes {
		if remote.Cron != "" && remote.URL != "" {
			logger.Debug("Adding cron job for whitelist", "url", remote.URL, "cron", remote.Cron)
			_, err := cronScheduler.AddFunc(remote.Cron, createWhitelistUpdateJob(cfg, logger, remote.URL, cacheInstance))
			if err != nil {
				logger.Error("Failed to add whitelist cron job: %v", "error", err)
			}
			logger.Debug("After adding whitelist job, entries count", "count", len(cronScheduler.Entries()))
		}
	}
}

// startSchedulerIfNeeded starts the cron scheduler if jobs are configured
func startSchedulerIfNeeded(cronScheduler *cron.Cron, logger *logging.Logger, cacheInstance *cache.Cache) {
	// Only start the scheduler if we have at least one scheduled job
	if len(cronScheduler.Entries()) > 0 {
		logger.Info("Starting cron scheduler with entries", "count", len(cronScheduler.Entries()))
		go cronScheduler.Start()
		// Note: cronScheduler will run until the program exits
		// No need to explicitly stop it as the program will terminate
	} else {
		logger.Debug("No cron jobs configured, scheduler not started")
	}
}

// createBlacklistUpdateJob creates a cron job function for blacklist updates
func createBlacklistUpdateJob(cfg *config.Config, logger *logging.Logger, url string, cacheInstance *cache.Cache) func() {
	return func() {
		logger.Info("Running scheduled update for remote blacklist", "url", url)
		// Load both local and remote blacklist files
		loadBlocklists(cfg, true)
		newMatcher := loadIPMatcher(cfg)
		if newMatcher != nil {
			cfg.IPMatcher = newMatcher
		}
		if cfg.Cache.AutoClearOnChange && cfg.Cache.Enabled {
			logger.Info("Auto-clearing cache due to scheduled blacklist update")
			if cacheInstance != nil {
				cacheInstance.Clear()
			}
		}
	}
}

// createWhitelistUpdateJob creates a cron job function for whitelist updates
func createWhitelistUpdateJob(cfg *config.Config, logger *logging.Logger, url string, cacheInstance *cache.Cache) func() {
	return func() {
		logger.Info("Running scheduled update for remote whitelist", "url", url)
		// Load both local and remote whitelist files
		loadWhitelists(cfg, true)
		newMatcher := loadIPMatcher(cfg)
		if newMatcher != nil {
			cfg.IPMatcher = newMatcher
		}
		if cfg.Cache.AutoClearOnChange && cfg.Cache.Enabled {
			logger.Info("Auto-clearing cache due to scheduled whitelist update")
			if cacheInstance != nil {
				cacheInstance.Clear()
			}
		}
	}
}

// parseCLIFlags parses command line arguments for config file and log level
func parseCLIFlags() (string, string) {
	configFile := parseConfigFlag()
	logLevel := parseLogLevelFlag()

	return configFile, logLevel
}

// parseConfigFlag parses the config file flag from command line arguments
func parseConfigFlag() string {
	configFile := ""

	for i, arg := range os.Args[1:] {
		if (arg == "-config" || arg == "--config") && len(os.Args) > i+2 {
			configFile = os.Args[i+2]
		} else if strings.HasPrefix(arg, "-config=") || strings.HasPrefix(arg, "--config=") {
			configFile = extractFlagValue(arg, "config")
		}
	}

	return configFile
}

// parseLogLevelFlag parses the log level flag from command line arguments
func parseLogLevelFlag() string {
	logLevel := "INFO"

	for i, arg := range os.Args[1:] {
		if (arg == "-log.level" || arg == "--log.level") && len(os.Args) > i+2 {
			logLevel = os.Args[i+2]
		} else if strings.HasPrefix(arg, "-log.level=") || strings.HasPrefix(arg, "--log.level=") {
			logLevel = extractFlagValue(arg, "log.level")
		}
	}

	return logLevel
}

// extractFlagValue extracts the value from a flag in the format -flag=value or --flag=value
func extractFlagValue(flag string, flagName string) string {
	parts := strings.SplitN(flag, "=", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
