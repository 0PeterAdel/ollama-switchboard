# Architecture

Switchboard is a local daemon exposing Ollama-compatible endpoints on localhost.
It selects local/cloud upstreams per model and policy, then retries cloud requests across a pool of upstream identities.
