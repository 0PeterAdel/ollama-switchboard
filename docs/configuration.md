# Configuration

`osb` currently uses **JSON** config (`config.json`).

See `examples/config.example.json`.

Notes:
- Durations accept strings (e.g. `"60s"`, `"300ms"`).
- `admin_token_required=true` requires either `security.admin_token` or `OSB_ADMIN_TOKEN`.
- `local_regex` and `cloud_regex` are validated at load time.
