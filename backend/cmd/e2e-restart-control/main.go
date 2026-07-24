// e2e-restart-control is a test-runner-only process supervisor. It is never
// registered by the production API and is copied only into the isolated E2E
// runner container.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type supervisor struct {
	mu         sync.Mutex
	serverPath string
	pidPath    string
	logFile    *os.File
	cmd        *exec.Cmd
}

type restartResponse struct {
	OldPID int `json:"oldPid"`
	NewPID int `json:"newPid"`
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: e2e-restart-control SERVER_BINARY SERVER_PID_FILE")
	}
	logPath := filepath.Join(filepath.Dir(os.Args[2]), "server.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	s := &supervisor{serverPath: os.Args[1], pidPath: os.Args[2], logFile: logFile}
	if _, err := s.start(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/restart", s.restart)
	log.Fatal(http.ListenAndServe("0.0.0.0:18090", nil))
}

func (s *supervisor) start() (int, error) {
	cmd := exec.Command(s.serverPath)
	cmd.Stdout, cmd.Stderr = s.logFile, s.logFile
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start server: %w", err)
	}
	if err := os.WriteFile(s.pidPath, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o644); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return 0, fmt.Errorf("write pid: %w", err)
	}
	s.cmd = cmd
	return cmd.Process.Pid, nil
}

func (s *supervisor) restart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil {
		http.Error(w, "server unavailable", http.StatusServiceUnavailable)
		return
	}
	pidBytes, err := os.ReadFile(s.pidPath)
	if err != nil {
		http.Error(w, "server pid file unavailable", http.StatusServiceUnavailable)
		return
	}
	oldPID, err := strconv.Atoi(string(bytes.TrimSpace(pidBytes)))
	if err != nil || oldPID != s.cmd.Process.Pid {
		http.Error(w, "server pid file does not identify supervised server", http.StatusInternalServerError)
		return
	}
	process, err := os.FindProcess(oldPID)
	if err != nil || process.Signal(syscall.SIGTERM) != nil {
		http.Error(w, "terminate failed", http.StatusInternalServerError)
		return
	}
	wait := make(chan error, 1)
	go func() { wait <- s.cmd.Wait() }()
	select {
	case <-time.After(10 * time.Second):
		_ = s.cmd.Process.Kill()
		<-wait
		http.Error(w, "server did not stop after SIGTERM", http.StatusGatewayTimeout)
		return
	case <-wait:
	}
	// Wait has reaped the exact child, so this is a durable process stop rather
	// than merely a port transition.
	if _, err := os.Stat("/proc/" + strconv.Itoa(oldPID)); !os.IsNotExist(err) {
		http.Error(w, "terminated server PID still exists", http.StatusInternalServerError)
		return
	}
	_ = os.Remove(s.pidPath)
	newPID, err := s.start()
	if err != nil {
		http.Error(w, "restart failed", http.StatusInternalServerError)
		return
	}
	if err := waitForServerReady(); err != nil {
		_ = s.cmd.Process.Signal(syscall.SIGTERM)
		_ = s.cmd.Wait()
		http.Error(w, "fresh server did not become ready", http.StatusGatewayTimeout)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(restartResponse{OldPID: oldPID, NewPID: newPID})
}

func waitForServerReady() error {
	client := &http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://127.0.0.1:18088/api/health")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server readiness timeout")
}
