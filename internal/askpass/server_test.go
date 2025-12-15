package askpass_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/idursun/jjui/internal/askpass"
	"github.com/tailscale/peercred"
	"golang.org/x/crypto/ssh"
)

func TestMain(m *testing.M) {
	// catch early the askpass subprocess
	s := askpass.NewUnstartedServer("JJUI_TEST")
	if s.IsSubprocess() {
		return
	}
	os.Exit(m.Run())
}

func TestServerNotListening(t *testing.T) {
	s := askpass.NewUnstartedServer("JJUI_TEST")
	// s.StartListening() is never called
	started, cancel, env := s.NewSubprocess("test")
	count := s.DebugSubprocessCount()
	if count != 0 { // listening was not called: subprocess should not be tracked
		t.Fatalf("count should be 0, got: %v", count)
	}

	if len(env) > 0 || env == nil {
		t.Fatalf("env should be an empty non-nil slice, got: %#v", env)
	}
	started(42)
	cancel()
	count = s.DebugSubprocessCount()
	if count != 0 {
		t.Fatalf("count should be 0, got: %v", count)
	}
}

func TestServerListening(t *testing.T) {
	s := askpass.NewUnstartedServer("JJUI_TEST")
	// important part
	if err := s.StartListening(); err != nil {
		if errors.Is(err, peercred.ErrNotImplemented) {
			t.Skip(err.Error())
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	started, cancel, env := s.NewSubprocess("test")
	count := s.DebugSubprocessCount()
	if count != 1 {
		t.Fatalf("count should be 1, got: %v", count)
	}

	if len(env) == 0 {
		t.Fatalf("env should not be empty, got: %v", env)
	}
	started(42)
	cancel()
	count = s.DebugSubprocessCount()
	if count != 0 {
		t.Fatalf("count should be 0, got: %v", count)
	}
}

func TestServerAskpass(t *testing.T) {
	expectedPassword := rand.Text()

	// ssh server to authenticate against
	var authOk atomic.Int32
	var authFail atomic.Int32
	addr, knownHost := startSSHServer(t, func(username, password string) error {
		if password != expectedPassword {
			if authFail.Add(1) > 2 {
				t.Log("unexpected password", password)
			}
			return ssh.ErrNoAuth
		}
		authOk.Add(1)
		return nil
	})
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}

	// main process, with user-facing password prompting
	s := askpass.NewUnstartedServer("JJUI_TEST")
	if err := s.StartListening(); err != nil {
		if errors.Is(err, peercred.ErrNotImplemented) {
			t.Skip(err.Error())
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	go s.Serve(func(name, prompt string, done <-chan struct{}) []byte {
		switch authFail.Load() {
		case 2:
			return []byte(expectedPassword)
		case 1:
			return []byte("wrong password")
		case 0:
			return nil // user denied
		default:
			panic(fmt.Sprintf("%d failure", authFail.Load()))
		}
	})

	// background ssh command
	started, cancel, env := s.NewSubprocess("ssh")
	defer cancel()

	cmd := exec.Command("ssh", "git@"+host, "-p", port,
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile="+knownHost,
		"-o", "PubkeyAuthentication=no",
		"-o", "PreferredAuthentications=password",
		"-o", "PKCS11Provider=none",
	)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("ssh client not available")
		}
		t.Fatal(err)
	}
	started(cmd.Process.Pid)
	_ = cmd.Wait() // ignore ssh session failure, we only care about the successful auth
	if n := authOk.Load(); n != 1 {
		t.Fatalf("auth did not happen exactly once: %v", n)
	}
	if n := authFail.Load(); n != 2 {
		t.Fatalf("auth failure did not happen exactly twice: %v", n)
	}
}

func startSSHServer(t *testing.T, auth func(username, password string) error) (addr, knownHostPath string) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			return nil, auth(conn.User(), string(password))
		},
		ServerVersion: "SSH-2.0-AskpassTest",
	}
	key, err := newHostKey()
	if err != nil {
		t.Fatal(err)
	}
	config.AddHostKey(key)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	knownHostPath = filepath.Join(t.TempDir(), "known_hosts")
	line := "127.0.0.1 " + string(ssh.MarshalAuthorizedKey(key.PublicKey()))
	err = os.WriteFile(knownHostPath, []byte(line), 0o666)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for t.Context().Err() == nil {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_, chans, reqs, err := ssh.NewServerConn(conn, config)
			if err != nil {
				conn.Close()
				continue
			}

			go ssh.DiscardRequests(reqs)
			go func() {
				for newChannel := range chans {
					newChannel.Reject(ssh.Prohibited, "only test the connection")
				}
			}()
			conn.Close()
		}
	}()
	return ln.Addr().String(), knownHostPath
}

func newHostKey() (ssh.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, err
	}
	return signer, nil
}
