# teams-quiet-hours

Install once, choose your work hours, and Teams stops disturbing you outside them.

`teams-quiet-hours` helps people set real work boundaries by silencing or blocking Microsoft Teams outside working hours without muting their whole machine.

Default schedule:

- Allow Teams Monday-Friday, 09:00-17:00
- Block before 09:00
- Block after 17:00
- Block all weekend

## Install

Users do not need Go installed and do not need to clone this repository.

Linux/macOS:

```sh
curl -fsSL https://github.com/OcheOps/teams-quiet-hours/releases/latest/download/install.sh | sh
```

Windows PowerShell as Administrator:

```powershell
iwr https://github.com/OcheOps/teams-quiet-hours/releases/latest/download/install.ps1 -UseB | iex
```

The installer downloads the correct prebuilt binary for your OS and CPU, verifies the release checksum when possible, installs the binary, then starts:

```sh
teams-quiet-hours setup
```

## Guided Setup

`setup` asks plain questions:

- What time should Teams start notifying you?
- What time should Teams stop notifying you?
- Should weekends be blocked?
- Which mode do you want?
- Which browser do you use?
- Should Teams reopen automatically during work hours?

Run setup again anytime:

```sh
sudo teams-quiet-hours setup
```

Windows users should run setup from an Administrator PowerShell window.

## Modes

### Soft

Default and safest.

- Blocks Teams web notifications through Chrome/Chromium/Edge enterprise browser policies.
- Leaves Teams desktop apps running.
- Does not close tabs.
- Does not block network access.

Soft mode blocks notifications only. It does not reliably stop Teams desktop calls/ringing.

### Hard

Opt-in.

- Blocks Teams web notifications.
- Closes/kills Teams desktop processes outside work hours.
- Does not block network access unless explicitly enabled.

Hard mode closes Teams outside work hours.

### Nuclear

Opt-in for stronger boundaries.

- Blocks Teams web notifications.
- Closes/kills Teams desktop processes.
- Can add experimental Teams network blocking when enabled.
- Installs repeated enforcement every few minutes.

Nuclear mode may interfere with Microsoft 365 workflows and should be used carefully.

## Everyday Commands

```sh
teams-quiet-hours status
teams-quiet-hours doctor
sudo teams-quiet-hours pause
sudo teams-quiet-hours resume
sudo teams-quiet-hours allow-now
sudo teams-quiet-hours block-now
sudo teams-quiet-hours uninstall
```

What they mean:

- `status`: show current configuration and enforcement state.
- `doctor`: friendly health check for permissions, browser policy, scheduler, Teams process, and network block state.
- `pause`: temporarily allow Teams and pause future enforcement.
- `resume`: resume quiet-hours enforcement.
- `allow-now`: immediately restore normal Teams behavior.
- `block-now`: immediately apply quiet-hours behavior.
- `uninstall`: remove the scheduler, config, installed binary, browser policies, and network rules created by this tool.

Advanced command:

```sh
sudo teams-quiet-hours enforce
```

`enforce` checks the current time and applies allow/block. It is safe for schedulers to run repeatedly.

## What Gets Installed

Binary:

- Linux/macOS: `/usr/local/bin/teams-quiet-hours`
- Windows: `%ProgramFiles%\teams-quiet-hours\teams-quiet-hours.exe`

Config:

- Linux: `/etc/teams-quiet-hours/config.json`
- macOS: `/Library/Application Support/teams-quiet-hours/config.json`
- Windows: `%ProgramData%\teams-quiet-hours\config.json`

Scheduler:

- Linux: systemd timers when available, cron fallback
- macOS: LaunchDaemons
- Windows: Task Scheduler tasks under `\teams-quiet-hours\`

Browser policies:

- Chrome/Chromium/Edge notification policy entries for Teams URLs only
- The tool never overwrites unrelated browser policies

Optional network blocking:

- Linux/macOS: a clearly marked `teams-quiet-hours` section in the hosts file
- Windows: only the firewall rule named `teams-quiet-hours Teams block`

## Release Assets

Each GitHub Release includes:

- `teams-quiet-hours-linux-amd64`
- `teams-quiet-hours-linux-arm64`
- `teams-quiet-hours-darwin-amd64`
- `teams-quiet-hours-darwin-arm64`
- `teams-quiet-hours-windows-amd64.exe`
- `teams-quiet-hours_<version>_amd64.deb`
- `teams-quiet-hours_<version>_arm64.deb`
- `install.sh`
- `install.ps1`
- `checksums.txt`

Ubuntu/Debian users can also download a `.deb` from Releases:

```sh
sudo dpkg -i teams-quiet-hours_<version>_amd64.deb
sudo teams-quiet-hours setup
```

## Security And Transparency

The tool is intentionally explicit about system changes:

- It manages only Teams notification URL policies.
- It records/removes only Windows registry values it creates.
- It creates/removes only scheduler entries named for `teams-quiet-hours`.
- It creates/removes only the firewall/hosts entries named for `teams-quiet-hours`.
- It always supports rollback with `teams-quiet-hours uninstall`.

Use `--dry-run` to preview changes:

```sh
sudo teams-quiet-hours block-now --mode nuclear --block-network --dry-run
```

Use `--verbose` with doctor for technical details:

```sh
teams-quiet-hours doctor --verbose
```

## Troubleshooting

Chrome/Chromium:

1. Open `chrome://policy`.
2. Click `Reload policies`.
3. Confirm `NotificationsBlockedForUrls` appears when blocked.
4. Restart the browser if needed.

Microsoft Edge:

1. Open `edge://policy`.
2. Click `Reload policies`.
3. Confirm `NotificationsBlockedForUrls` appears when blocked.

If desktop calls still ring in soft mode, use hard mode. Browser notification policies do not reliably control desktop app calls.

If nuclear mode affects more than intended:

```sh
sudo teams-quiet-hours allow-now
sudo teams-quiet-hours uninstall
```

## Packaging Roadmap

Available now:

- Prebuilt Linux/macOS/Windows binaries
- GitHub Releases with checksums
- `install.sh` for Linux/macOS
- `install.ps1` for Windows
- `.deb` packages for Ubuntu/Debian

Planned later:

- `.rpm` package for Fedora/RHEL
- Optional APT repository
- Homebrew tap
- macOS `.pkg` installer
- Windows MSI or winget manifest

## Product Backlog

This project is solving a real work-boundary problem: many people do not hate work itself, they hate the inability to leave work. Future work should keep that human framing: work boundaries without muting your whole machine.

Near-term polish:

- Temporary override: allow Teams for `15m`, `30m`, `1h`, or until tomorrow.
- Tray icon or small GUI with clear state: Teams allowed, quiet hours active, paused.
- One-click actions: pause 1 hour, allow until tomorrow, open settings.
- Smarter setup: detect installed browsers, Teams desktop app, timezone, and likely browser-only vs desktop-app usage.
- `teams-quiet-hours explain`: show exactly what policies, schedulers, firewall rules, hosts entries, and process controls exist.

Focus and availability ideas:

- Presence/status automation: set Teams to Offline, Away, or Do Not Disturb outside work hours, then restore previous status during work hours.
- Meeting exceptions: allow scheduled meeting reminders, calls from favorites, or calendar-based temporary allow windows.
- Deep work modes: silence Teams when fullscreen apps, screen sharing, OBS, Zoom, or coding workflows are active.
- Multiple work profiles: separate schedules for work laptop, personal machine, or different browser profiles.
- Browser profile targeting: block Teams only in a specific Chrome/Edge profile such as Work, Company, or Corp.

Longer-term platform ideas:

- Calendar integration with Outlook or Google Calendar.
- Homebrew tap, RPM package, AUR package, MSI, winget, and optional package repositories.
- Enterprise/team mode for healthy-boundary policies, labor-law compliance, timezone-aware schedules, and managed exceptions.

Branding direction:

- Keep the tone calm and human, not hostile to work.
- Avoid framing the tool as a blocker, killer, or anti-work hack.
- Strong positioning: "Work boundaries without muting your whole machine."
- Possible future names or related product language: AfterHours, QuietTeams, ClockOut, WorkBoundary, TeamSilence, OfficeHours, Unping, PeaceMode.

V1 restraint:

- Do not over-engineer the first stable version.
- The most valuable V1 is install, setup wizard, schedule, Teams notification blocking, optional Teams app closing, tray status, and temporary override.
- Firewall rules, calendar integrations, browser-profile introspection, and Teams API automation should stay clearly marked as advanced/future work until they are safe and polished.

## Development

```sh
make fmt
make test
make lint
make cross-build
make package-deb
```

The project uses only the Go standard library.

## License

MIT
