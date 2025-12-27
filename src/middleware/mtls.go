package middleware

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
)

// MTLSConfig holds mTLS configuration
type MTLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
	Enabled  bool
}

// CreateTLSConfig creates a TLS config with mTLS settings
func CreateTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// Load CA certificate for client verification
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config with client certificate verification
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
		MinVersion: tls.VersionTLS13, // Use TLS 1.3 for better performance
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	return tlsConfig, nil
}

// MTLSMiddleware validates client certificates
func MTLSMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if TLS connection exists
			if c.Request().TLS == nil {
				return echo.NewHTTPError(401, "TLS required")
			}

			// Verify client certificate was provided
			if len(c.Request().TLS.PeerCertificates) == 0 {
				return echo.NewHTTPError(401, "Client certificate required")
			}

			// Client certificate is valid (verified by TLS handshake)
			// Additional checks can be added here (e.g., CN validation)

			return next(c)
		}
	}
}
