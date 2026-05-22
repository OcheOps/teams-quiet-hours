# teams-quiet-hours

`teams-quiet-hours` is a small Linux CLI utility that blocks Microsoft Teams web notifications in Google Chrome outside configured work hours, without muting all browser or system notifications.

It works by managing one Chrome Enterprise policy file:

```text
/etc/opt/chrome/policies/managed/teams-quiet-hours.json
```

When quiet hours are active, the tool writes the `NotificationsBlockedForUrls` policy for Teams URLs. When notifications should be allowed, it removes only that one managed policy file.

## Default Schedule

- Allow Teams notifications Monday-Friday, 09:00-17:00
- Block Teams notifications before 09:00
- Block Teams notifications after 17:00
- Block Teams notifications on Saturday and Sunday

## Install

From a checkout:

```sh
sudo ./bin/teams-quiet-hours install --timezone Africa/Lagos
```

This installs:

- `/usr/local/bin/teams-quiet-hours`
- `/etc/cron.d/teams-quiet-hours`
- `/etc/teams-quiet-hours/config`

The cron file runs:

- `allow` at 09:00 Monday-Friday
- `block` at 17:00 Monday-Friday
- `block` at 00:00 Saturday and Sunday

## Usage

```sh
teams-quiet-hours install
teams-quiet-hours uninstall
teams-quiet-hours block
teams-quiet-hours allow
teams-quiet-hours status
teams-quiet-hours config
```

Commands that write under `/etc` or `/usr/local/bin` require root:

```sh
sudo teams-quiet-hours block
sudo teams-quiet-hours allow
```

Check what would be changed without writing files:

```sh
sudo teams-quiet-hours install --dry-run
sudo teams-quiet-hours block --dry-run
```

## Browser Targets

Google Chrome is the default:

```sh
sudo teams-quiet-hours install --browser chrome
```

Chromium paths are also supported:

```sh
sudo teams-quiet-hours install --browser chromium
```

This manages both common Chromium policy directories:

```text
/etc/chromium/policies/managed/
/etc/chromium-browser/policies/managed/
```

To manage Chrome and Chromium together:

```sh
sudo teams-quiet-hours install --browser all
```

## Custom Hours

Use 24-hour `HH:MM` times:

```sh
sudo teams-quiet-hours install --start 08:30 --end 18:00 --timezone Africa/Lagos
```

The timezone is written as `CRON_TZ` in `/etc/cron.d/teams-quiet-hours`. If omitted, cron uses the system timezone.

## Policy File

When blocked, Chrome receives:

```json
{
  "NotificationsBlockedForUrls": [
    "https://teams.microsoft.com/*",
    "https://*.teams.microsoft.com/*"
  ]
}
```

When allowed, `teams-quiet-hours` removes only:

```text
teams-quiet-hours.json
```

It never edits unrelated Chrome policy files.

## Uninstall

```sh
sudo teams-quiet-hours uninstall
```

This removes the managed policy file, cron file, installed CLI, and config file created by the tool.

## Troubleshooting

1. Open `chrome://policy`.
2. Click `Reload policies`.
3. Confirm `NotificationsBlockedForUrls` appears when blocked.
4. Restart Chrome if policies do not update immediately.
5. Run `teams-quiet-hours status` to confirm which policy files exist.

If `block` fails with a JSON validation error, install either `jq` or `python3`.

## Development

Run the validation script:

```sh
./tests/run.sh
```

Run ShellCheck if installed:

```sh
shellcheck bin/teams-quiet-hours tests/run.sh
```

The tests redirect system paths into a temporary directory and do not require root.

## Packaging

`.deb` packaging is intentionally left for a later version. The current project is a single Bash script plus cron/config files, so it can be packaged with `fpm`, `dpkg-deb`, or a native Debian packaging directory later.

## License

MIT
