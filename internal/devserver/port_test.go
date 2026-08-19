package devserver

import (
	"errors"
	"net"
	"reflect"
	"testing"
)

type fakeListener struct {
	address net.Addr
	closed  bool
}

func (listener *fakeListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (listener *fakeListener) Addr() net.Addr            { return listener.address }
func (listener *fakeListener) Close() error {
	listener.closed = true
	return nil
}

func TestListenTriesCommonPortsThenKeepsOSEphemeralListener(t *testing.T) {
	var addresses []string
	wantListener := &fakeListener{address: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321}}
	listen := func(_, address string) (net.Listener, error) {
		addresses = append(addresses, address)
		if address == "127.0.0.1:0" {
			return wantListener, nil
		}
		return nil, errors.New("busy")
	}

	listener, port, err := Listen("127.0.0.1", 0, false, listen)
	if err != nil {
		t.Fatal(err)
	}
	if listener != wantListener || port != 54321 || wantListener.closed {
		t.Fatalf("listener = %v, port = %d, closed = %v", listener, port, wantListener.closed)
	}
	wantAddresses := []string{
		"127.0.0.1:8080",
		"127.0.0.1:8000",
		"127.0.0.1:3000",
		"127.0.0.1:1313",
		"127.0.0.1:4000",
		"127.0.0.1:0",
	}
	if !reflect.DeepEqual(addresses, wantAddresses) {
		t.Fatalf("addresses = %v, want %v", addresses, wantAddresses)
	}
}

func TestListenUsesOnlyExplicitPort(t *testing.T) {
	wantErr := errors.New("occupied")
	var addresses []string
	listen := func(_, address string) (net.Listener, error) {
		addresses = append(addresses, address)
		return nil, wantErr
	}

	_, _, err := Listen("127.0.0.1", 4444, true, listen)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if want := []string{"127.0.0.1:4444"}; !reflect.DeepEqual(addresses, want) {
		t.Fatalf("addresses = %v, want %v", addresses, want)
	}
}

func TestListenRejectsInvalidExplicitPort(t *testing.T) {
	called := false
	listen := func(_, _ string) (net.Listener, error) {
		called = true
		return nil, nil
	}

	for _, port := range []int{0, -1, 65536} {
		if _, _, err := Listen("127.0.0.1", port, true, listen); err == nil {
			t.Fatalf("port %d accepted", port)
		}
	}
	if called {
		t.Fatal("listener called for invalid explicit port")
	}
}

func TestURLPreservesIPv6AndBasePath(t *testing.T) {
	if got, want := URL("::1", 8080, "/docs/"), "http://[::1]:8080/docs/"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
	if got, want := URL("127.0.0.1", 8000, ""), "http://127.0.0.1:8000/"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}
