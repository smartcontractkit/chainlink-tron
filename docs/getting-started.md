# Chainlin Integration - TRON - Getting started

1. Install Nix following instructions [here](./../.misc/dev-guides/nix/getting-started.md).
2. Enter developer environment
3. Continue with the [local setup](./run-local.md)

## Developer environment

TRON scripts require `docker`, `psql`, and other pkgs.

Enter the developer environment using Nix:

```bash
nix develop .#tron
```

## Packages

List all available developer shells and packages with:

```bash
nix flake show
```

## Package - TRON LOOP plugin

### Build

Build `chainlink-tron` Nix package:

```bash
nix build .#chainlink-tron --print-out-paths
```

Build `chainlink-tron` Nix package without checking out the source code locally:

```bash
nix build 'git+ssh://git@github.com/smartcontractkit/chainlink-integrations'#chainlink-tron --print-out-paths
```

**Notice (issue):** there is a known issue with packaging Go 1.23 aplications with gomod2nix which is referenced in [relayer/default.nix](../../relayer/default.nix) that has to do with `vendor/modules.txt` specification and new requirements. While the issue is getting resolved we package and publish a `chainlink-tron-build` build script which can be ran manually to produce a build output.

Build `chainlink-tron` bin manually using the provided script:

```bash
# Move to the main relayer project and build
nix run .#chainlink-tron-build # will output ./result/bin/chainlink-tron compiled binary
```

Build `chainlink-tron` bin manually:

```bash
# Enter the TRON default dev shell
nix develop .#tron
# Move to the main relayer project and build
cd tron/relayer
go build
```
