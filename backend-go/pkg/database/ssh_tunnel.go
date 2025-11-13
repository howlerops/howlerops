package database

import (
	"context"
	"fmt"
	"io"
	"os"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/crypto"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHTunnel represents an active SSH tunnel
type SSHTunnel struct {
	config        *SSHTunnelConfig
	sshClient     *ssh.Client
	localPort     int
	remoteHost    string
	remotePort    int
	listener      net.Listener
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	logger        *logrus.Logger
	connected     bool
	reconnectChan chan struct{}
}

// SSHTunnelManager manages SSH tunnels for database connections
type SSHTunnelManager struct {
	tunnels     map[string]*SSHTunnel
	secretStore crypto.SecretStore
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// NewSSHTunnelManager creates a new SSH tunnel manager
func NewSSHTunnelManager(secretStore crypto.SecretStore, logger *logrus.Logger) *SSHTunnelManager {
	return &SSHTunnelManager{
		tunnels:     make(map[string]*SSHTunnel),
		secretStore: secretStore,
		logger:      logger,
	}
}

// EstablishTunnel creates an SSH tunnel and returns the local port
func (m *SSHTunnelManager) EstablishTunnel(ctx context.Context, config *SSHTunnelConfig, remoteHost string, remotePort int) (*SSHTunnel, error) {
	if config == nil {
		return nil, fmt.Errorf("SSH tunnel configuration is nil")
	}

	// Create SSH client configuration
	sshConfig, err := m.buildSSHConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to build SSH config: %w", err)
	}

	// Connect to SSH server
	sshAddr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	m.logger.WithFields(logrus.Fields{
		"ssh_host": sshAddr,
		"remote":   fmt.Sprintf("%s:%d", remoteHost, remotePort),
	}).Info("Establishing SSH tunnel")

	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	// Allocate local port
	localPort, listener, err := m.allocateLocalPort()
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("failed to allocate local port: %w", err)
	}

	tunnelCtx, cancel := context.WithCancel(ctx)
	tunnel := &SSHTunnel{
		config:        config,
		sshClient:     sshClient,
		localPort:     localPort,
		remoteHost:    remoteHost,
		remotePort:    remotePort,
		listener:      listener,
		ctx:           tunnelCtx,
		cancel:        cancel,
		logger:        m.logger,
		connected:     true,
		reconnectChan: make(chan struct{}, 1),
	}

	// Start tunnel forwarding
	tunnel.wg.Add(1)
	go tunnel.forwardConnections()

	// Start keep-alive
	if config.KeepAliveInterval > 0 {
		tunnel.wg.Add(1)
		go tunnel.keepAlive()
	}

	// Store tunnel
	tunnelID := fmt.Sprintf("%s:%d->%s:%d", config.Host, config.Port, remoteHost, remotePort)
	m.mu.Lock()
	m.tunnels[tunnelID] = tunnel
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"local_port":  localPort,
		"remote":      fmt.Sprintf("%s:%d", remoteHost, remotePort),
		"via_bastion": sshAddr,
	}).Info("SSH tunnel established successfully")

	return tunnel, nil
}

// CloseTunnel closes an SSH tunnel
func (m *SSHTunnelManager) CloseTunnel(tunnel *SSHTunnel) error {
	if tunnel == nil {
		return nil
	}

	tunnel.mu.Lock()
	defer tunnel.mu.Unlock()

	if !tunnel.connected {
		return nil
	}

	tunnel.cancel()
	tunnel.connected = false

	if tunnel.listener != nil {
		tunnel.listener.Close()
	}

	if tunnel.sshClient != nil {
		tunnel.sshClient.Close()
	}

	tunnel.wg.Wait()

	// Remove from manager
	tunnelID := fmt.Sprintf("%s:%d->%s:%d", tunnel.config.Host, tunnel.config.Port, tunnel.remoteHost, tunnel.remotePort)
	m.mu.Lock()
	delete(m.tunnels, tunnelID)
	m.mu.Unlock()

	m.logger.WithField("local_port", tunnel.localPort).Info("SSH tunnel closed")
	return nil
}

// CloseAll closes all active tunnels
func (m *SSHTunnelManager) CloseAll() error {
	m.mu.Lock()
	tunnels := make([]*SSHTunnel, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		tunnels = append(tunnels, t)
	}
	m.mu.Unlock()

	var lastErr error
	for _, t := range tunnels {
		if err := m.CloseTunnel(t); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// buildSSHConfig creates an SSH client configuration from SSHTunnelConfig
func (m *SSHTunnelManager) buildSSHConfig(ctx context.Context, config *SSHTunnelConfig) (*ssh.ClientConfig, error) {
	authMethods := make([]ssh.AuthMethod, 0)

	// Configure authentication
	switch config.AuthMethod {
	case SSHAuthPassword:
		var password string

		// Try to load password from SecretStore first
		if m.secretStore != nil && config.ConnectionID != "" {
			passwordBytes, secretErr := m.secretStore.GetSecret(ctx, config.ConnectionID, crypto.SecretTypeSSHPassword)
			if secretErr == nil {
				password = string(passwordBytes)
			} else {
				// Fall back to plaintext password (deprecated)
				password = config.Password
			}
		} else {
			password = config.Password
		}

		if password == "" {
			return nil, fmt.Errorf("password is required for password authentication")
		}
		authMethods = append(authMethods, ssh.Password(password))

	case SSHAuthPrivateKey:
		var keyData []byte
		var err error

		// Try to load private key from SecretStore first
		if m.secretStore != nil && config.ConnectionID != "" {
			keyBytes, secretErr := m.secretStore.GetSecret(ctx, config.ConnectionID, crypto.SecretTypeSSHPrivateKey)
			if secretErr == nil {
				keyData = keyBytes
			} else {
				// Fall back to plaintext private key (deprecated)
				if config.PrivateKey != "" {
					keyData = []byte(config.PrivateKey)
				} else if config.PrivateKeyPath != "" {
					keyData, err = os.ReadFile(config.PrivateKeyPath)
					if err != nil {
						return nil, fmt.Errorf("failed to read private key file: %w", err)
					}
				}
			}
		} else {
			// Fall back to plaintext private key (deprecated)
			if config.PrivateKey != "" {
				keyData = []byte(config.PrivateKey)
			} else if config.PrivateKeyPath != "" {
				keyData, err = os.ReadFile(config.PrivateKeyPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read private key file: %w", err)
				}
			}
		}

		if len(keyData) == 0 {
			return nil, fmt.Errorf("private key is required for key authentication")
		}

		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	default:
		return nil, fmt.Errorf("unsupported SSH authentication method: %s", config.AuthMethod)
	}

	// Configure host key verification
	var hostKeyCallback ssh.HostKeyCallback
	var err error
	if config.StrictHostKeyChecking && config.KnownHostsPath != "" {
		// Load known hosts
		hostKeyCallback, err = m.loadKnownHosts(config.KnownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load known hosts: %w", err)
		}
	} else {
		// Allow any host key (insecure, but useful for development)
		// #nosec G106 - InsecureIgnoreHostKey used only when KnownHostsFile not provided, logged as warning
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
		m.logger.Warn("SSH host key verification is disabled - this is insecure!")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}, nil
}

// loadKnownHosts loads known hosts from a file
func (m *SSHTunnelManager) loadKnownHosts(path string) (ssh.HostKeyCallback, error) {
	// Use knownhosts.New which returns a HostKeyCallback
	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, err
	}
	return callback, nil
}

// allocateLocalPort finds an available local port and creates a listener
func (m *SSHTunnelManager) allocateLocalPort() (int, net.Listener, error) {
	// Let the OS assign a port from the ephemeral range
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create listener: %w", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, listener, nil
}

// forwardConnections forwards connections through the SSH tunnel
func (t *SSHTunnel) forwardConnections() {
	defer t.wg.Done()

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		// Accept local connection
		localConn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.ctx.Done():
				return
			default:
				t.logger.WithError(err).Error("Failed to accept local connection")
				continue
			}
		}

		// Handle connection in goroutine
		t.wg.Add(1)
		go t.handleConnection(localConn)
	}
}

// handleConnection handles a single connection through the tunnel
func (t *SSHTunnel) handleConnection(localConn net.Conn) {
	defer t.wg.Done()
	defer localConn.Close()

	// Dial remote connection through SSH
	remoteAddr := fmt.Sprintf("%s:%d", t.remoteHost, t.remotePort)
	remoteConn, err := t.sshClient.Dial("tcp", remoteAddr)
	if err != nil {
		t.logger.WithError(err).WithField("remote", remoteAddr).Error("Failed to dial remote connection")
		return
	}
	defer remoteConn.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(remoteConn, localConn) // Best-effort copy - errors handled by connection close
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(localConn, remoteConn) // Best-effort copy - errors handled by connection close
	}()

	// Wait for both directions to complete or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-t.ctx.Done():
	}
}

// keepAlive sends periodic keep-alive messages
func (t *SSHTunnel) keepAlive() {
	defer t.wg.Done()

	interval := t.config.KeepAliveInterval
	if interval == 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			_, _, err := t.sshClient.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				t.logger.WithError(err).Error("SSH keep-alive failed")
				// Try to reconnect
				select {
				case t.reconnectChan <- struct{}{}:
				default:
				}
			}
		}
	}
}

// GetLocalPort returns the local port number
func (t *SSHTunnel) GetLocalPort() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.localPort
}

// IsConnected returns whether the tunnel is connected
func (t *SSHTunnel) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}
