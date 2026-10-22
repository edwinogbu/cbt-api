//go:build windows

// cbt-windows-service wraps cbt-api-server.exe as a native Windows
// service, so a School CBT Node's API server starts automatically on
// boot, restarts on crash (via the service's own recovery policy, set
// at install time), and is controllable with the standard Windows
// Services console / `sc.exe` / `net start|stop` - no Docker, no WSL2,
// no third-party service-wrapper tool (NSSM etc.) required. This
// process never contains any application logic itself: it only starts,
// supervises, and stops the real cbt-api-server.exe binary (expected to
// sit next to this one), the same binary `go build ./cmd/server/`
// already produces and that already runs correctly under Docker on
// Linux - nothing about the application changes between the two.
//
// Install (run once, from an elevated/Administrator PowerShell, in the
// directory containing both this .exe and cbt-api-server.exe):
//
//	cbt-windows-service.exe install
//	net start CBTSchoolNode
//
// Remove:
//
//	net stop CBTSchoolNode
//	cbt-windows-service.exe remove
//
// Configuration: a `service.env` file (KEY=VALUE per line, # comments
// allowed) in the same directory as this .exe - the same environment
// variables documented in .env.school-node.example, since this wraps
// the identical application. DATABASE_URL, REDIS_ADDR, JWT_SECRET,
// JWT_REFRESH_SECRET, and ENCRYPTION_MASTER_KEY are required; the
// server refuses to start without them, same as on Linux.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "CBTSchoolNode"
const serviceDisplayName = "CBT School Node API Server"
const serviceDescription = "Runs this school's local CBT API server (database, exams, question bank) entirely on the school's own network, with no internet dependency."

func main() {
	if len(os.Args) > 1 {
		switch strings.ToLower(os.Args[1]) {
		case "install":
			exitOnErr(installService())
			return
		case "remove":
			exitOnErr(removeService())
			return
		case "run", "debug":
			// Run in the foreground, logging to stdout - for local
			// testing before installing as an actual service.
			exitOnErr(runForeground())
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q - use install, remove, or run\n", os.Args[1])
			os.Exit(2)
		}
	}

	// No arguments: this is how the Windows Service Control Manager
	// actually launches an installed service.
	isService, err := svc.IsWindowsService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to determine if running as a Windows service: %v\n", err)
		os.Exit(1)
	}
	if !isService {
		fmt.Println("Not running as a Windows service and no command given - use install, remove, or run.")
		os.Exit(1)
	}
	exitOnErr(svc.Run(serviceName, &cbtService{}))
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// serverExePath resolves cbt-api-server.exe relative to this service
// binary's own directory, never relative to the SCM's working
// directory (which is unpredictable - typically C:\Windows\System32).
func serverExePath() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve own executable path: %w", err)
	}
	dir := filepath.Dir(self)
	exe := filepath.Join(dir, "cbt-api-server.exe")
	if _, err := os.Stat(exe); err != nil {
		return "", fmt.Errorf("cbt-api-server.exe not found next to %s - both must be installed in the same directory: %w", self, err)
	}
	return exe, nil
}

// loadServiceEnv reads KEY=VALUE pairs from service.env (next to this
// binary) into a []string suitable for exec.Cmd.Env - never modifies
// this process's own environment, so the service wrapper's env stays
// predictable even if service.env is edited and the service restarted.
func loadServiceEnv(dir string) ([]string, error) {
	env := os.Environ()
	path := filepath.Join(dir, "service.env")
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return env, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			continue
		}
		env = append(env, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return env, nil
}

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to the Windows Service Control Manager (run as Administrator): %w", err)
	}
	defer m.Disconnect()

	if s, err := m.OpenService(serviceName); err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists - run 'remove' first to reinstall", serviceName)
	}

	s, err := m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Automatic restart on crash - a submitted exam must not stay down
	// because the process happened to die, and a school has no on-call
	// engineer to manually restart it.
	err = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 15 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400) // reset the failure count after a day of stability
	if err != nil {
		// Non-fatal - the service itself is installed and will still
		// run; just without automatic crash recovery configured.
		fmt.Fprintf(os.Stderr, "warning: service installed, but failed to configure crash-recovery actions: %v\n", err)
	}

	if err := eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to register event log source: %v\n", err)
	}

	fmt.Printf("Service %s installed. Start it with: net start %s\n", serviceName, serviceName)
	return nil
}

func removeService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to the Windows Service Control Manager (run as Administrator): %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed: %w", serviceName, err)
	}
	defer s.Close()

	if err := s.Delete(); err != nil {
		return fmt.Errorf("failed to remove service: %w", err)
	}
	_ = eventlog.Remove(serviceName)
	fmt.Printf("Service %s removed.\n", serviceName)
	return nil
}

// runForeground runs the same supervise loop as the installed service,
// but in the current console with logs on stdout - for verifying a
// deployment works before registering it as an actual Windows service.
func runForeground() error {
	exe, err := serverExePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(exe)
	env, err := loadServiceEnv(dir)
	if err != nil {
		return err
	}

	cmd := exec.Command(exe)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Starting %s ...\n", exe)
	return cmd.Run()
}

type cbtService struct{}

func (s *cbtService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	elog, err := eventlog.Open(serviceName)
	if err == nil {
		defer elog.Close()
	}
	logf := func(format string, a ...interface{}) {
		msg := fmt.Sprintf(format, a...)
		if elog != nil {
			_ = elog.Info(1, msg)
		}
	}

	changes <- svc.Status{State: svc.StartPending}

	exe, err := serverExePath()
	if err != nil {
		logf("failed to locate cbt-api-server.exe: %v", err)
		return false, 1
	}
	dir := filepath.Dir(exe)
	env, err := loadServiceEnv(dir)
	if err != nil {
		logf("failed to load service.env: %v", err)
		return false, 1
	}

	logFile, logErr := os.OpenFile(filepath.Join(dir, "cbt-api-service.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logErr != nil {
		// Not fatal - the server can still run without a log file, it
		// just won't have one. Continuing here (rather than failing the
		// whole service) matches this service's own job: keep the API
		// server running, not gate it on logging infrastructure.
		logf("warning: failed to open log file: %v", logErr)
	}

	cmd := exec.Command(exe)
	cmd.Dir = dir
	cmd.Env = env
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		defer logFile.Close()
	}

	if err := cmd.Start(); err != nil {
		logf("failed to start cbt-api-server.exe: %v", err)
		return false, 1
	}
	logf("cbt-api-server.exe started (pid %d)", cmd.Process.Pid)

	// Reap the child process's exit in the background - if it crashes,
	// this service stops itself too, rather than reporting "running" to
	// the SCM while actually serving nothing; the SCM's own recovery
	// actions (configured at install time) then restart THIS service,
	// which spawns a fresh child.
	childDone := make(chan error, 1)
	go func() { childDone <- cmd.Wait() }()

	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	logf("service running")

loop:
	for {
		select {
		case err := <-childDone:
			logf("cbt-api-server.exe exited unexpectedly: %v", err)
			return false, 1
		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				changes <- req.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				logf("stopping cbt-api-server.exe (pid %d)", cmd.Process.Pid)
				_ = cmd.Process.Kill()
				<-childDone
				break loop
			}
		}
	}

	changes <- svc.Status{State: svc.Stopped}
	return false, 0
}
