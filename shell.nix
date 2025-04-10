{
  stdenv,
  pkgs,
  lib,
}:
# juno requires building with clang, not gcc
pkgs.mkShell {
  buildInputs = with pkgs;
    [
      # nix tooling
      alejandra

      # Go 1.23 + tools
      go_1_23
      gopls
      delve
      golangci-lint
      gotools
      gomod2nix
      # Official golang implementation of the Ethereum protocol (e.g., geth, abigen, rlpdump, etc.)
      go-ethereum

      # Extra tools
      git
      python3
      postgresql_15
      jq
    ]
    ++ lib.optionals stdenv.hostPlatform.isDarwin [
      libiconv
    ];
}
