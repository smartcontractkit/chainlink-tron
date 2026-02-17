{
  stdenv,
  pkgs,
  lib,
}:
pkgs.mkShell {
  buildInputs = with pkgs;
    [
      # nix tooling
      alejandra

      # Prefer Go 1.25 (required by chainlink-common), fallback for older nixpkgs channels.
      (if pkgs ? go_1_25 then go_1_25 else go_1_24)
      gopls
      delve
      golangci-lint
      gotools
      go-mockery_2

      # Extra tools
      git
      jq
      kubectl
      kubernetes-helm
    ]
    ++ lib.optionals stdenv.hostPlatform.isDarwin [
      libiconv
    ];
}
