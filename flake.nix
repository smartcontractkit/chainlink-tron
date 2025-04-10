{
  description = "Chainlink TRON - a repository of Chainlink integration components to support TRON";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/release-24.11";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
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
        overlays = [
          inputs.gomod2nix.overlays.default
        ];
      };

      # The rev (git commit hash) of the current flake
      rev = self.rev or self.dirtyRev or "-";

      # The common arguments to pass to the packages
      commonArgs = {
        inherit pkgs;
        inherit rev;
      };
      #
      #   # TODO: Resolve subprojects
      #   relayer = pkgs.callPackage ./relayer commonArgs;
    in rec {
      # Output a set of dev environments (shells)
      devShells = {
        default = pkgs.callPackage ./shell.nix {inherit pkgs;};
      };

      #   # TODO: Output a set of packages (e.g., CL core node plugins, sc artifacts, etc.)
      #   packages =
      #     {}
      #     // relayer.packages;
    });
}
