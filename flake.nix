{
  description = "Chainlink TRON - a repository of Chainlink integration components to support TRON";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = inputs @ {
    self,
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      # Import nixpkgs with specific configuration
      pkgs = import nixpkgs {
        inherit system;
      };

      # The rev (git commit hash) of the current flake
      rev = self.rev or self.dirtyRev or "-";

      # The common arguments to pass to the packages
      commonArgs = {
        inherit pkgs;
        inherit rev;
      };

      # Resolve root module
      chainlink-tron = pkgs.callPackage ./relayer/cmd/chainlink-tron commonArgs;
      # Resolve sub-modules
      # TODO: package EVM contracts from source
      # contracts = pkgs.callPackage ./contracts commonArgs;
      contracts = {
        devShells = {};
        packages = {};
      };
    in rec {
      # Output a set of dev environments (shells)
      devShells =
        {
          default = pkgs.callPackage ./shell.nix {inherit pkgs;};
        }
        // contracts.devShells;

      # Output a set of packages (e.g., CL core node plugins, sc artifacts, etc.)
      packages =
        {
          # Chainlink core node plugin (default + alias)
          inherit chainlink-tron;
          default = chainlink-tron;
        }
        // contracts.packages;
    });
}
