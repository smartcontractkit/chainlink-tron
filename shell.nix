{
  stdenv,
  pkgs,
  lib,
}:
let
  # Force mockery to be built with Go 1.25 so it can parse modules requiring go 1.25+.
  mockery =
    if pkgs ? buildGo125Module
    then pkgs.go-mockery_2.override {buildGoModule = pkgs.buildGo125Module;}
    else pkgs.go-mockery_2;
in
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
      mockery

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
