package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/OcheOps/teams-quiet-hours/internal/config"
	"github.com/OcheOps/teams-quiet-hours/internal/enforcer"
	"github.com/OcheOps/teams-quiet-hours/internal/platform"
)

var version = "0.4.0"

func usage() {
	fmt.Fprintf(os.Stderr, `teams-quiet-hours - quiet Microsoft Teams browser notifications outside work hours

Usage:
  teams-quiet-hours setup
  teams-quiet-hours install [--mode soft|hard|nuclear] [--browser chrome|chromium|edge|all] [--start HH:MM] [--end HH:MM] [--timezone TZ] [--dry-run]
  teams-quiet-hours uninstall [--browser chrome|chromium|edge|all] [--dry-run]
  teams-quiet-hours block-now [--mode soft|hard|nuclear] [--browser chrome|chromium|edge|all] [--dry-run]
  teams-quiet-hours allow-now [--browser chrome|chromium|edge|all] [--dry-run]
  teams-quiet-hours enforce [--dry-run]
  teams-quiet-hours status [--browser chrome|chromium|edge|all]
  teams-quiet-hours pause
  teams-quiet-hours resume
  teams-quiet-hours doctor [--verbose]
  teams-quiet-hours config
  teams-quiet-hours start [--dry-run]
  teams-quiet-hours stop [--dry-run]

Commands:
  setup       Guided first-time setup.
  install     Install binary, config, and OS scheduler.
  uninstall   Remove scheduler, config, installed binary, and this tool's policy entries.
  block-now   Apply quiet-hours behavior immediately.
  allow-now   Restore normal Teams behavior immediately.
  enforce     Check the current schedule and apply allow/block.
  pause       Temporarily pause quiet-hours enforcement.
  resume      Resume quiet-hours enforcement.
  doctor      Check permissions, scheduler, browser policy, Teams, and rollback state.
  status      Show config, scheduler, and policy state.
  config      Print active configuration.
  start       Launch Teams/web Teams without changing policy.
  stop        Stop Teams desktop processes without changing policy.

Options:
  --mode      soft, hard, or nuclear. Default: soft.
  --browser   chrome, chromium, edge, or all. Default: chrome.
  --start     Weekday allow start in HH:MM. Default: 09:00.
  --end       Weekday allow end in HH:MM. Default: 17:00.
  --timezone  Scheduler timezone where supported. Linux cron supports CRON_TZ.
  --weekdays  Allowed work days. Default: mon-fri.
  --launch-on-allow      Launch Teams/web Teams when allowing.
  --close-browser-tabs   Attempt Teams browser tab closing where supported.
  --block-network        Opt in to experimental Teams network blocking.
  --dry-run   Print planned writes without changing files.
  --verbose   Show extra technical detail for troubleshooting.
  --version   Print version.
`)
}

type cliArgs struct {
	Command string
	Config  config.Config
	DryRun  bool
	Verbose bool
}

func parseArgs(args []string) (cliArgs, error) {
	if len(args) == 0 {
		return cliArgs{}, fmt.Errorf("missing command")
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Printf("teams-quiet-hours %s\n", version)
		os.Exit(0)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		usage()
		os.Exit(0)
	}

	command := args[0]
	switch command {
	case "setup", "install", "uninstall", "block", "allow", "block-now", "allow-now", "status", "enforce", "pause", "resume", "doctor", "config", "start", "stop":
	default:
		return cliArgs{}, fmt.Errorf("unknown command %q", command)
	}

	cfg, err := config.Load()
	if err != nil {
		return cliArgs{}, err
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	opts := config.Options{}
	fs.StringVar(&opts.Mode, "mode", "", "quiet-hours mode")
	fs.StringVar(&opts.Browser, "browser", "", "browser target")
	fs.StringVar(&opts.Start, "start", "", "weekday start time")
	fs.StringVar(&opts.End, "end", "", "weekday end time")
	fs.StringVar(&opts.Timezone, "timezone", "", "scheduler timezone")
	fs.StringVar(&opts.Weekdays, "weekdays", "", "allowed weekdays")
	fs.BoolFunc("launch-on-allow", "launch Teams when allowing", func(value string) error {
		opts.LaunchOnAllow = value != "false"
		opts.LaunchOnAllowSet = true
		return nil
	})
	fs.BoolFunc("close-browser-tabs", "close Teams browser tabs where supported", func(value string) error {
		opts.CloseBrowserTabs = value != "false"
		opts.CloseTabsSet = true
		return nil
	})
	fs.BoolFunc("block-network", "enable experimental network blocking", func(value string) error {
		opts.BlockNetwork = value != "false"
		opts.BlockNetworkSet = true
		return nil
	})
	fs.BoolVar(&opts.DryRun, "dry-run", false, "dry run")
	verbose := fs.Bool("verbose", false, "verbose output")
	if err := fs.Parse(args[1:]); err != nil {
		return cliArgs{}, err
	}
	cfg = config.Merge(cfg, opts)
	if err := config.Validate(cfg); err != nil {
		return cliArgs{}, err
	}
	return cliArgs{Command: command, Config: cfg, DryRun: opts.DryRun, Verbose: *verbose}, nil
}

func requireAdmin(dryRun bool) error {
	if dryRun {
		return nil
	}
	if os.Getenv("TEAMS_QUIET_HOURS_SYSTEM_PREFIX") != "" {
		return nil
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		if os.Geteuid() != 0 {
			return fmt.Errorf("this command writes system policy/scheduler files and must be run with sudo/root")
		}
	case "windows":
		// A non-admin user normally cannot enumerate local sessions.
		if err := runQuiet("net", "session"); err != nil {
			return fmt.Errorf("this command writes HKLM policies and scheduled tasks; run from an Administrator terminal")
		}
	}
	return nil
}

func hasAdmin() bool {
	if os.Getenv("TEAMS_QUIET_HOURS_SYSTEM_PREFIX") != "" {
		return true
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		return os.Geteuid() == 0
	case "windows":
		return runQuiet("net", "session") == nil
	default:
		return false
	}
}

func runQuiet(name string, args ...string) error {
	return nilIfExecUnavailable(name, args...)
}

func installPath() string {
	if override := os.Getenv("TEAMS_QUIET_HOURS_BIN_PATH"); override != "" {
		return override
	}
	if prefix := os.Getenv("TEAMS_QUIET_HOURS_SYSTEM_PREFIX"); prefix != "" {
		if runtime.GOOS == "windows" {
			return filepath.Join(prefix, "Program Files", "teams-quiet-hours", "teams-quiet-hours.exe")
		}
		return filepath.Join(prefix, "usr", "local", "bin", "teams-quiet-hours")
	}
	switch runtime.GOOS {
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		return filepath.Join(programFiles, "teams-quiet-hours", "teams-quiet-hours.exe")
	default:
		return "/usr/local/bin/teams-quiet-hours"
	}
}

func copySelf(target string, dryRun bool) error {
	source, err := os.Executable()
	if err != nil {
		return err
	}
	sourceAbs, _ := filepath.Abs(source)
	targetAbs, _ := filepath.Abs(target)
	if sourceAbs == targetAbs {
		if dryRun {
			fmt.Printf("DRY-RUN: binary already installed at %s\n", target)
		}
		return nil
	}
	if dryRun {
		fmt.Printf("DRY-RUN: install binary %s -> %s\n", source, target)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func removeInstalledBinary(path string, dryRun bool) error {
	if dryRun {
		fmt.Printf("DRY-RUN: remove %s\n", path)
		return nil
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func printConfig(cfg config.Config) {
	fmt.Printf("Platform: %s\n", platform.Name())
	fmt.Printf("Mode: %s\n", cfg.Mode)
	fmt.Printf("Browser: %s\n", cfg.Browser)
	fmt.Printf("Weekday allow start: %s\n", cfg.Start)
	fmt.Printf("Weekday allow end: %s\n", cfg.End)
	fmt.Printf("Weekdays: %s\n", cfg.Weekdays)
	fmt.Printf("Launch on allow: %t\n", cfg.LaunchOnAllow)
	fmt.Printf("Close browser tabs: %t\n", cfg.CloseBrowserTabs)
	fmt.Printf("Block network: %t\n", cfg.BlockNetwork)
	fmt.Printf("Paused: %t\n", cfg.Paused)
	if cfg.Timezone == "" {
		fmt.Println("Timezone: scheduler default")
	} else {
		fmt.Printf("Timezone: %s\n", cfg.Timezone)
	}
	fmt.Printf("Config file: %s\n", config.Path())
}

func prompt(reader *bufio.Reader, question string, fallback string) string {
	fmt.Printf("%s [%s]: ", question, fallback)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return fallback
	}
	return answer
}

func promptBool(reader *bufio.Reader, question string, fallback bool) bool {
	defaultText := "n"
	if fallback {
		defaultText = "y"
	}
	answer := strings.ToLower(prompt(reader, question+" (y/n)", defaultText))
	return answer == "y" || answer == "yes"
}

func promptChoice(reader *bufio.Reader, question string, fallback string, allowed map[string]string) string {
	for {
		answer := strings.ToLower(prompt(reader, question, fallback))
		if value, ok := allowed[answer]; ok {
			return value
		}
		fmt.Println("Please choose one of the listed options.")
	}
}

func runSetup(rt platform.Runtime, ef enforcer.Enforcer, current config.Config, dryRun bool) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("teams-quiet-hours setup")
	fmt.Println("Install once, choose your work hours, and Teams stops disturbing you outside them.")
	fmt.Println()

	cfg := current
	cfg.Start = prompt(reader, "What time should Teams start notifying you?", cfg.Start)
	cfg.End = prompt(reader, "What time should Teams stop notifying you?", cfg.End)
	if promptBool(reader, "Should weekends be blocked?", true) {
		cfg.Weekdays = "mon-fri"
	} else {
		cfg.Weekdays = "sun,mon,tue,wed,thu,fri,sat"
	}
	cfg.Mode = promptChoice(reader, "Choose mode: soft, hard, or nuclear", cfg.Mode, map[string]string{
		"soft": "soft", "hard": "hard", "nuclear": "nuclear",
	})
	cfg.Browser = promptChoice(reader, "Which browser do you use: chrome, edge, chromium, or all", cfg.Browser, map[string]string{
		"chrome": "chrome", "edge": "edge", "chromium": "chromium", "all": "all",
	})
	cfg.LaunchOnAllow = promptBool(reader, "Should Teams reopen automatically during work hours?", cfg.LaunchOnAllow)
	if cfg.Mode == "nuclear" {
		cfg.BlockNetwork = promptBool(reader, "Enable experimental Teams network blocking?", false)
	}
	cfg.Paused = false

	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Setup could not continue: %s\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("This will save your quiet-hours settings, install the scheduler, and apply the current state.")
	if cfg.Mode == "hard" || cfg.Mode == "nuclear" {
		fmt.Printf("%s mode can close Teams outside work hours.\n", cfg.Mode)
	}
	if !promptBool(reader, "Apply this setup now?", true) {
		fmt.Println("Setup cancelled. Nothing was changed.")
		return
	}

	target := installPath()
	if err := copySelf(target, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing binary: %s\n", err)
		os.Exit(1)
	}
	if err := config.Save(cfg, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %s\n", err)
		os.Exit(1)
	}
	if err := rt.Scheduler.Install(cfg, target, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing scheduler: %s\n", err)
		os.Exit(1)
	}
	if err := ef.Enforce(cfg, time.Now(), dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error applying current quiet-hours state: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Setup complete. Run 'teams-quiet-hours status' anytime.")
}

func runDoctor(rt platform.Runtime, cfg config.Config, verbose bool) {
	fmt.Println("teams-quiet-hours doctor")
	fmt.Println()
	if hasAdmin() {
		fmt.Println("Permissions: Administrator/root access is available.")
	} else {
		fmt.Println("Permissions: Administrator/root access is not active. Setup, uninstall, pause/resume, allow-now, and block-now may need elevation.")
	}
	printConfig(cfg)
	fmt.Println()

	schedulerStatuses, err := rt.Scheduler.Status()
	if err != nil {
		fmt.Printf("Scheduler: needs attention (%s)\n", err)
	} else {
		for _, status := range schedulerStatuses {
			fmt.Printf("Scheduler %s: installed=%t\n", status.Name, status.Installed)
			if verbose {
				fmt.Printf("  %s: %s\n", status.Target, status.Detail)
			}
		}
	}
	policyStatuses, err := rt.Policy.Status(cfg.Browser)
	if err != nil {
		fmt.Printf("Browser policy: needs attention (%s)\n", err)
	} else {
		for _, status := range policyStatuses {
			fmt.Printf("Browser policy %s: blocked=%t\n", status.Browser, status.Blocked)
			if verbose {
				fmt.Printf("  %s: %s\n", status.Target, status.Detail)
			}
		}
	}
	processStatuses, err := rt.Process.Status()
	if err != nil {
		fmt.Printf("Teams app: could not check (%s)\n", err)
	} else {
		for _, status := range processStatuses {
			fmt.Printf("Teams process %s: active=%t\n", status.Target, status.Enabled)
		}
	}
	networkStatuses, err := rt.Network.Status()
	if err != nil {
		fmt.Printf("Network block: could not check (%s)\n", err)
	} else {
		for _, status := range networkStatuses {
			fmt.Printf("Network block: active=%t\n", status.Enabled)
			if verbose {
				fmt.Printf("  %s: %s\n", status.Target, status.Detail)
			}
		}
	}
	fmt.Println()
	fmt.Println("Rollback: run 'teams-quiet-hours uninstall' from an Administrator/root terminal.")
}

func main() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		usage()
		os.Exit(1)
	}

	rt := currentRuntime()
	ef := enforcer.Enforcer{Runtime: rt}
	writeCommand := args.Command == "setup" || args.Command == "install" || args.Command == "uninstall" || args.Command == "block" || args.Command == "allow" || args.Command == "block-now" || args.Command == "allow-now" || args.Command == "enforce" || args.Command == "pause" || args.Command == "resume" || args.Command == "stop"
	if writeCommand {
		if err := requireAdmin(args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	}

	switch args.Command {
	case "setup":
		runSetup(rt, ef, args.Config, args.DryRun)
	case "install":
		target := installPath()
		if err := copySelf(target, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing binary: %s\n", err)
			os.Exit(1)
		}
		if err := config.Save(args.Config, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %s\n", err)
			os.Exit(1)
		}
		if err := rt.Scheduler.Install(args.Config, target, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing scheduler: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Installed teams-quiet-hours for %s at %s\n", rt.Name, target)
		fmt.Printf("Schedule: allow %s at %s-%s, block outside that window\n", args.Config.Weekdays, args.Config.Start, args.Config.End)
		if args.Config.Mode == "hard" || args.Config.Mode == "nuclear" {
			fmt.Printf("Warning: %s mode can close Teams outside work hours.\n", args.Config.Mode)
		}
	case "uninstall":
		if err := ef.Allow(args.Config, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing policy: %s\n", err)
			os.Exit(1)
		}
		if err := rt.Scheduler.Uninstall(args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing scheduler: %s\n", err)
			os.Exit(1)
		}
		if err := config.Remove(args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing config: %s\n", err)
			os.Exit(1)
		}
		if err := removeInstalledBinary(installPath(), args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove installed binary: %s\n", err)
		}
		fmt.Println("Uninstalled teams-quiet-hours")
	case "block", "block-now":
		if err := ef.Block(args.Config, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error blocking Teams: %s\n", err)
			os.Exit(1)
		}
	case "allow", "allow-now":
		if err := ef.Allow(args.Config, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error allowing Teams: %s\n", err)
			os.Exit(1)
		}
	case "pause":
		args.Config.Paused = true
		if err := config.Save(args.Config, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error pausing teams-quiet-hours: %s\n", err)
			os.Exit(1)
		}
		if err := ef.Allow(args.Config, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error allowing Teams while paused: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("teams-quiet-hours is paused. Run 'teams-quiet-hours resume' to re-enable it.")
	case "resume":
		args.Config.Paused = false
		if err := config.Save(args.Config, args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error resuming teams-quiet-hours: %s\n", err)
			os.Exit(1)
		}
		if err := ef.Enforce(args.Config, time.Now(), args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying current quiet-hours state: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("teams-quiet-hours is resumed.")
	case "enforce":
		if err := ef.Enforce(args.Config, time.Now(), args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error enforcing Teams quiet hours: %s\n", err)
			os.Exit(1)
		}
	case "status":
		printConfig(args.Config)
		fmt.Printf("Install target: %s\n", installPath())
		schedulerStatuses, err := rt.Scheduler.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading scheduler status: %s\n", err)
			os.Exit(1)
		}
		for _, status := range schedulerStatuses {
			fmt.Printf("Scheduler %s: installed=%t target=%s detail=%s\n", status.Name, status.Installed, status.Target, status.Detail)
		}
		policyStatuses, err := rt.Policy.Status(args.Config.Browser)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading policy status: %s\n", err)
			os.Exit(1)
		}
		for _, status := range policyStatuses {
			fmt.Printf("Policy %s: blocked=%t target=%s detail=%s\n", status.Browser, status.Blocked, status.Target, status.Detail)
		}
		processStatuses, err := rt.Process.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading Teams process status: %s\n", err)
			os.Exit(1)
		}
		for _, status := range processStatuses {
			fmt.Printf("Teams %s: active=%t target=%s detail=%s\n", status.Name, status.Enabled, status.Target, status.Detail)
		}
		networkStatuses, err := rt.Network.Status()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading Teams network status: %s\n", err)
			os.Exit(1)
		}
		for _, status := range networkStatuses {
			fmt.Printf("Teams %s: active=%t target=%s detail=%s\n", status.Name, status.Enabled, status.Target, status.Detail)
		}
	case "config":
		printConfig(args.Config)
	case "doctor":
		runDoctor(rt, args.Config, args.Verbose)
	case "start":
		if err := rt.Process.Start(args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting Teams: %s\n", err)
			os.Exit(1)
		}
	case "stop":
		if err := rt.Process.Stop(args.DryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping Teams: %s\n", err)
			os.Exit(1)
		}
	}
}
