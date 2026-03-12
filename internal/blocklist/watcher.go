package blocklist

import (
	"time"

	"github.com/dombyte/ipgate/internal/logging"
	"github.com/fsnotify/fsnotify"
)

// WatchBlocklists watches files for changes and triggers a callback when changes are detected
func WatchBlocklists(paths []string, callback func(), logger *logging.Logger) error {
	// Create and configure watcher
	watcher, err := createWatcher()
	if err != nil {
		logError(logger, "Failed to create file watcher", err)
		return err
	}

	// Add paths to watcher
	if err := addPathsToWatcher(watcher, paths); err != nil {
		return err
	}

	// Start watching in goroutine
	startWatcher(watcher, callback, logger)

	return nil
}

// createWatcher creates a new file watcher instance
func createWatcher() (*fsnotify.Watcher, error) {
	return fsnotify.NewWatcher()
}

// addPathsToWatcher adds paths to the watcher
func addPathsToWatcher(watcher *fsnotify.Watcher, paths []string) error {
	for _, path := range paths {
		if err := watcher.Add(path); err != nil {
			return err
		}
	}
	return nil
}

// startWatcher starts the watcher goroutine
func startWatcher(watcher *fsnotify.Watcher, callback func(), logger *logging.Logger) {
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Handle file change event
				handleFileChangeEvent(event, callback, logger)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// Handle watcher error
				logError(logger, "Watcher error", err)
			}
		}
	}()
}

// handleFileChangeEvent handles a file change event
func handleFileChangeEvent(event fsnotify.Event, callback func(), logger *logging.Logger) {
	// Check if event is a file change
	if isFileChangeEvent(event) {
		logFileChange(logger, event)
		// Add small delay to handle multiple events from single change
		time.Sleep(100 * time.Millisecond)
		callback()
	}
}

// isFileChangeEvent checks if the event indicates a file change
func isFileChangeEvent(event fsnotify.Event) bool {
	return event.Op&fsnotify.Write == fsnotify.Write ||
		event.Op&fsnotify.Create == fsnotify.Create ||
		event.Op&fsnotify.Remove == fsnotify.Remove ||
		event.Op&fsnotify.Rename == fsnotify.Rename ||
		event.Op&fsnotify.Chmod == fsnotify.Chmod
}

// logFileChange logs a file change event
func logFileChange(logger *logging.Logger, event fsnotify.Event) {
	if logger != nil {
		logger.Info("Detected change in %s (event: %s), reloading...", "file", event.Name, "event", event.Op)
	}
}

// logError logs an error if logger is available
func logError(logger *logging.Logger, message string, err error) {
	if logger != nil {
		logger.Error(message+": %v", "error", err)
	}
}
