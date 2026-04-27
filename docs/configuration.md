# Configuration

`osb` loads JSON and YAML configuration files. The default path is `config.json` in the platform config directory.

Examples:

- `examples/config.example.json`
- `examples/config.example.yaml`

Durations can be strings such as `"60s"`, `"300ms"`, or `"2m"`. Integer duration values are treated as nanoseconds for compatibility with Go's `time.Duration` JSON encoding.

## Routing

Supported policies:

- `auto`: route by suffix and regex rules, otherwise local.
- `local-only`: always use the local upstream.
- `cloud-only`: always use a configured cloud upstream.
- `prefer-local`: try local first, then fall back to cloud.
- `prefer-cloud`: try cloud first, then fall back to local.

`cloud_suffix`, `cloud_regex`, and `local_regex` are validated when config is loaded. Cloud suffix and regex rules take precedence over local regex rules.

## Admin API

When `security.admin_token_required` is `true`, set either `security.admin_token` or the `OSB_ADMIN_TOKEN` environment variable. Admin requests must include `X-OSB-Admin-Token: <token>` or `Authorization: Bearer <token>`.

## Runtime Updates

`osb add`, `osb remove`, and `osb reload` sync upstream changes to the running daemon through the admin API. Listener addresses, routing policy, and other daemon-wide settings still require a restart.
