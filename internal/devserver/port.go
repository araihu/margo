// Package devserver implements Margo's development-only site server.
package devserver

import (
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
)

var automaticPorts = [...]int{8080, 8000, 3000, 1313, 4000}

// ListenFunc opens one network listener.
type ListenFunc func(network, address string) (net.Listener, error)

// Listen selects and retains a listener for the development server.
func Listen(host string, port int, explicit bool, listen ListenFunc) (net.Listener, int, error) {
	if strings.TrimSpace(host) == "" {
		return nil, 0, fmt.Errorf("serve.host_invalid: host is required")
	}
	if listen == nil {
		listen = net.Listen
	}
	if explicit {
		if port < 1 || port > 65535 {
			return nil, 0, fmt.Errorf("serve.port_invalid: port must be between 1 and 65535")
		}
		listener, err := listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return nil, 0, fmt.Errorf("serve.bind_failed: %s:%d: %w", host, port, err)
		}
		return listener, port, nil
	}

	for _, candidate := range automaticPorts {
		listener, err := listen("tcp", net.JoinHostPort(host, strconv.Itoa(candidate)))
		if err == nil {
			return listener, candidate, nil
		}
	}
	listener, err := listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return nil, 0, fmt.Errorf("serve.bind_failed: no automatic development port is available: %w", err)
	}
	selected, err := listenerPort(listener)
	if err != nil {
		_ = listener.Close()
		return nil, 0, err
	}
	return listener, selected, nil
}

func listenerPort(listener net.Listener) (int, error) {
	if listener == nil || listener.Addr() == nil {
		return 0, fmt.Errorf("serve.listener_invalid: listener has no address")
	}
	if address, ok := listener.Addr().(*net.TCPAddr); ok && address.Port > 0 {
		return address.Port, nil
	}
	_, value, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return 0, fmt.Errorf("serve.listener_invalid: %w", err)
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("serve.listener_invalid: listener reported port %q", value)
	}
	return port, nil
}

// URL returns the browser URL for a selected listener and site base path.
func URL(host string, port int, basePath string) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" || basePath == "/" {
		basePath = "/"
	} else {
		basePath = path.Clean("/"+strings.Trim(basePath, "/")) + "/"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + basePath
}
