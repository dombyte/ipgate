// Package blocklist provides functionality for loading and managing IP blocklists.
// It supports both local and remote file sources, with validation and
// high-performance IP matching using the bart radix tree library.
//
// Key features:
// - Local file loading with size validation
// - Remote file loading with HTTP client connection pooling
// - IPv4/IPv6 classification and counting
// - High-performance IP matching using bart radix trees
// - Legacy compatibility for backward compatibility
package blocklist

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dombyte/ipgate/internal/ipmatcher"
)

// httpClient is a shared HTTP client with connection pooling
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// getHTTPClient returns a shared HTTP client with connection pooling configuration.
// This function uses a sync.Once pattern to ensure the HTTP client is created
// only once, even when called from multiple goroutines.
//
// The HTTP client is configured with:
// - Connection pooling (100 max idle connections)
// - Keep-alive support
// - 90-second idle connection timeout
// - 30-second request timeout
//
// Returns:
//
//	*http.Client - Shared HTTP client with connection pooling
//
// This function is thread-safe and can be called concurrently.
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			MaxIdleConns:        10000,
			MaxIdleConnsPerHost: 10000,
			MaxConnsPerHost:     10000,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	})
	return httpClient
}

// LoadRemoteFile loads IP entries from a remote URL.
// It uses a shared HTTP client with connection pooling for performance
// and fetches the file with size limits to prevent memory exhaustion.
//
// The function:
// - Fetches the file from the specified URL
// - Validates HTTP status code
// - Limits response size to prevent memory exhaustion
// - Parses the file line by line
// - Skips empty lines and comments (lines starting with #)
// - Returns all valid IP entries
//
// Parameters:
//
//	url - URL to fetch the remote file from
//	maxSize - Maximum allowed size for the response body
//
// Returns:
//
//	[]string - List of IP entries found in the file
//	error - Any error that occurred during fetching or parsing
//
// Example:
//
//	entries, err := LoadRemoteFile("https://example.com/blocklist.txt", 10485760)
func LoadRemoteFile(url string, maxSize int64) ([]string, error) {
	client := getHTTPClient()

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status %d", url, resp.StatusCode)
	}

	// Limit the response body size
	limitedReader := io.LimitedReader{R: resp.Body, N: maxSize + 1}
	var entries []string
	scanner := bufio.NewScanner(&limitedReader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			entries = append(entries, line)
		}
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("error reading response from %s: %v", url, scanner.Err())
	}

	if limitedReader.N <= 0 {
		return nil, fmt.Errorf("response from %s exceeds max size (%d bytes)", url, maxSize)
	}

	return entries, nil
}

// LoadLocalFile loads IP entries from a local file.
// It validates file size before reading to prevent memory exhaustion
// and parses the file line by line.
//
// The function:
// - Opens and validates the file exists
// - Checks file size against maximum limit
// - Parses the file line by line
// - Skips empty lines and comments (lines starting with #)
// - Returns all valid IP entries
//
// Parameters:
//
//	path - Filesystem path to the local file
//	maxSize - Maximum allowed size for the file
//
// Returns:
//
//	[]string - List of IP entries found in the file
//	error - Any error that occurred during opening, reading, or parsing
//
// Example:
//
//	entries, err := LoadLocalFile("/path/to/blocklist.txt", 10485760)
func LoadLocalFile(path string, maxSize int64) ([]string, error) {
	// Use os.Root to scope file access under current directory, preventing directory traversal
	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, fmt.Errorf("failed to open root: %v", err)
	}
	file, err := root.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", path, err)
	}
	defer file.Close()

	stat, err := root.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %v", path, err)
	}
	if stat.Size() > maxSize {
		return nil, fmt.Errorf("file %s exceeds max size (%d bytes)", path, maxSize)
	}

	var entries []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			entries = append(entries, line)
		}
	}
	return entries, scanner.Err()
}

// IsIPBlocked checks if an IP is blocked or whitelisted using the IPMatcher.
// This is the new implementation using bart for high-performance matching.
//
// The function:
// - Validates the IPMatcher is not nil
// - Delegates to the IPMatcher's IsBlocked method
// - Returns whether the IP is blocked, the reason, and any error
//
// Parameters:
//
//	ipStr - The IP address to check (IPv4 or IPv6)
//	matcher - The IPMatcher instance containing blocklist and whitelist data
//
// Returns:
//
//	bool - true if the IP is blocked, false if allowed
//	string - Reason for the decision (e.g., "whitelisted", "matched IP")
//	error - Any error that occurred during checking
//
// Example:
//
//	blocked, reason, err := IsIPBlocked("192.168.1.1", matcher)
func IsIPBlocked(ipStr string, matcher *ipmatcher.IPMatcher) (bool, string, error) {
	if matcher == nil {
		return false, "", fmt.Errorf("IPMatcher is nil")
	}
	return matcher.IsBlocked(ipStr)
}
