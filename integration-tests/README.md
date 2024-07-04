# Integration Tests for Tron

## Running the tests locally with G++ and Nix

0. Go into the `integration-tests` directory

1. Set the SSH private key environment variable
   `export SSH_PRIVATE_KEY="$(cat /path/to/your/ssh/private/key)"`

2. Run the Integrations tests

```bash
PRIVATE_KEY=$SSH_PRIVATE_KEY DEBUG=* LOG_LEVEL=debug nix run --max-jobs 1 '.#integration-tests' --show-trace --print-build-logs
```
