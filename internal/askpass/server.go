// Package askpass provides a backchannel to handle ssh password prompts from the calling instance.
package askpass

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tailscale/peercred"
)

// NewUnstartedServer creates an unstarted server to handle askpass prompts.
func NewUnstartedServer(envPrefix string) *Server {
	return &Server{
		envPrefix:  envPrefix,
		socketPath: filepath.Join(os.TempDir(), strings.ToLower(envPrefix)+"-ssh-askpass-"+strconv.Itoa(os.Getpid())+".sock"),

		subprocesses: make(map[string]subprocess),
	}
}

type Server struct {
	envPrefix  string
	socketPath string

	ln atomic.Pointer[net.UnixListener]

	mu           sync.Mutex
	subprocesses map[string]subprocess
}

// IsSubprocess returns true if it detects that it was started as a SSH_ASKPASS subprocess. In this case the main program should shutdown immediately (stdout handling already happened before returning).
func (s *Server) IsSubprocess() bool {
	addr := os.Getenv(s.envPrefix + "_SSH_ASKPASS_ADDR")
	if addr == "" {
		return false
	}
	if err := dialServer(addr, os.Getenv(s.envPrefix+"_SSH_ASKPASS_KEY")); err != nil {
		log.Fatal(err)
	}
	return true
}

func dialServer(addr, key string) error {
	conn, err := net.Dial("unix", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	msg := key + "\n"
	if len(os.Args) > 1 {
		msg += os.Args[1]
	}
	msg += "\n"
	_, err = conn.Write([]byte(msg))
	if err != nil {
		return err
	}

	// check if the prompt was successful
	one := make([]byte, 1)
	n, err := conn.Read(one)
	if n == 0 || err != nil {
		return errors.New("user declined providing the password")
	}
	// forward the retrieved password to stdout
	_, err = os.Stdout.ReadFrom(conn)
	return err
}

// StartListening starts listening on the unix socket (must be called before [Server.Serve])
func (s *Server) StartListening() error {
	ln, err := net.ListenUnix("unix", &net.UnixAddr{
		Name: s.socketPath,
		Net:  "unix",
	})
	if err != nil {
		return err
	}

	s.ln.Store(ln)

	if err := s.smokeTestPeerCred(ln); err != nil {
		s.Close()
		return err
	}

	return nil
}

// check that "ensureConnFromDescendant" works with a self-established connection
func (s *Server) smokeTestPeerCred(ln *net.UnixListener) error {
	go func() {
		conn, err := net.Dial("unix", s.socketPath)
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Read(make([]byte, 1)) // darwin needs an active connection for peercred
	}()
	conn, err := ln.AcceptUnix()
	if err != nil {
		return err
	}
	defer conn.Close()
	// use ppid to check the readPPid logic as well
	return ensureConnFromDescendant(conn, os.Getppid())
}

// Close stops listening on the unix socket (and removes the file)
func (s *Server) Close() error {
	ln := s.ln.Swap(nil)
	if ln == nil {
		return nil
	}
	return errors.Join(
		ln.Close(),
		os.Remove(s.socketPath),
	)
}

// Serve calls askpass to retrieve the password and give it to askpass (must be called after [Server.StartListening])
// It will return an error only if the unix socket returns an error on ln.Accept.
//   - name comes from [Server.NewSubprocess]
//   - prompt is the prompt line provided by askpass
//   - done will be closed when the parent process returned (hence the password is no longer needed)
func (s *Server) Serve(askpass func(name, prompt string, done <-chan struct{}) []byte) error {
	ln := s.ln.Load()

	for {
		conn, err := ln.AcceptUnix()
		if err != nil {
			return err
		}
		go func() {
			err := s.handle(conn, askpass)
			if err != nil {
				log.Println("askpass connection failed:", err)
			}
		}()
	}
}

func (s *Server) handle(conn *net.UnixConn, askpass func(name, prompt string, done <-chan struct{}) []byte) error {
	defer conn.Close()

	// the read phase should be quick
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return err
	}
	scan := bufio.NewScanner(conn)
	if !scan.Scan() {
		return errors.Join(scan.Err(), errors.New("expected key"))
	}
	key := scan.Text()

	s.mu.Lock()
	sub, ok := s.subprocesses[key]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown key %s", key)
	}

	select {
	case <-sub.started:
		if err := ensureConnFromDescendant(conn, *sub.pid); err != nil {
			return err
		}
	case <-sub.done:
		return nil
	}

	if !scan.Scan() {
		return errors.Join(scan.Err(), errors.New("expected prompt"))
	}

	// wait for the user input
	pass := askpass(sub.name, scan.Text(), sub.done)
	if pass == nil {
		// user did not submit a password: close the connection
		return nil
	}
	defer clear(pass)
	select {
	case <-sub.done:
		return nil
	default:
	}

	// set a new deadline (the askpass callback may have taken a while)
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return err
	}
	_, err := conn.Write(append([]byte{' '}, pass...)) // leading byte means user provided a password
	return err
}

// NewSubprocess indicates the intent to start a subprocess which might need a password.
//   - started must be called with the pid of a parent process of the askpass invocation (to ensure that the password is only given to proper processes)
//   - cancel must be called when the subprocess is done
//   - env contains the env variables for the askpass override (never nil)
func (s *Server) NewSubprocess(name string) (started func(ppid int), cancel func(), env []string) {
	if s.ln.Load() == nil {
		return func(pid int) {}, func() {}, []string{}
	}

	key := rand.Text()
	startedCh := make(chan struct{})
	pid := new(int)
	done := make(chan struct{})
	sub := subprocess{
		name:    name,
		started: startedCh,
		pid:     pid,
		done:    done,
	}
	s.mu.Lock()
	s.subprocesses[key] = sub
	s.mu.Unlock()
	return func(ppid int) {
			*pid = ppid
			close(sub.started)
		}, func() {
			close(done)
			s.mu.Lock()
			delete(s.subprocesses, key)
			s.mu.Unlock()
		}, []string{
			"SSH_ASKPASS=" + os.Args[0],
			"SSH_ASKPASS_REQUIRE=force",
			s.envPrefix + "_SSH_ASKPASS_ADDR=" + s.socketPath,
			s.envPrefix + "_SSH_ASKPASS_KEY=" + key,
		}
}

type subprocess struct {
	name    string
	started chan struct{}
	pid     *int
	done    <-chan struct{}
}

func ensureConnFromDescendant(conn *net.UnixConn, parentPID int) error {
	cred, err := peercred.Get(conn)
	if err != nil {
		return err
	}
	pid, ok := cred.PID()
	if !ok || pid == 0 {
		return peercred.ErrNotImplemented
	}

	for pid != parentPID {
		pid, err = getPPid(pid)
		if err != nil {
			return err
		}
		if pid == 0 {
			pid, _ = cred.PID()
			return fmt.Errorf("PID %d is not a descendant of PID %d", pid, parentPID)
		}
	}
	return nil
}
