package util

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mattn/go-ieproxy"
)

// ProxyInfo holds information about detected proxy settings
type ProxyInfo struct {
	URL    string // Proxy URL (e.g., "http://proxy.example.com:8080")
	Source string // Where the proxy was detected from
}

// DetectProxy attempts to detect system proxy settings
// Returns proxy info if found, or nil if no proxy is configured
func DetectProxy() *ProxyInfo {
	// First check environment variables (highest priority)
	if envProxy := checkEnvProxy(); envProxy != nil {
		return envProxy
	}

	// Then check system proxy settings
	if sysProxy := checkSystemProxy(); sysProxy != nil {
		return sysProxy
	}

	return nil
}

// checkEnvProxy checks for proxy environment variables
func checkEnvProxy() *ProxyInfo {
	// Check common proxy environment variables
	envVars := []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"}

	for _, envVar := range envVars {
		if proxy := os.Getenv(envVar); proxy != "" {
			// Ensure the proxy URL has a scheme
			if !strings.HasPrefix(proxy, "http://") && !strings.HasPrefix(proxy, "https://") {
				proxy = "http://" + proxy
			}
			return &ProxyInfo{
				URL:    proxy,
				Source: fmt.Sprintf("environment variable %s", envVar),
			}
		}
	}

	return nil
}

// checkSystemProxy checks system proxy settings using ieproxy
func checkSystemProxy() *ProxyInfo {
	// Get proxy configuration from system
	conf := ieproxy.GetConf()
	if conf.Static.Active {
		// Static proxy is configured
		if conf.Static.Protocols["http"] != "" {
			return &ProxyInfo{
				URL:    conf.Static.Protocols["http"],
				Source: "system proxy settings",
			}
		}
		if conf.Static.Protocols["https"] != "" {
			return &ProxyInfo{
				URL:    conf.Static.Protocols["https"],
				Source: "system proxy settings",
			}
		}
	}

	return nil
}

// CreateHTTPTransportWithProxy creates an HTTP transport that respects proxy settings
func CreateHTTPTransportWithProxy() *http.Transport {
	// Start with a clone of the default transport
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// Use ieproxy to handle all proxy detection automatically
	// This will check environment variables first, then system settings
	transport.Proxy = ieproxy.GetProxyFunc()

	return transport
}

// GetProxyInfoForDisplay returns proxy information for display purposes
func GetProxyInfoForDisplay() *ProxyInfo {
	return DetectProxy()
}
