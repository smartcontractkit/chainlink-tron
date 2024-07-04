{
  description = "Chainlink Tron E2E Testing";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    # G++ package (CLI+Plugins) on Tron-Plugin branch
    gauntlet.url = "git+ssh://git@github.com/smartcontractkit/gauntlet-plus-plus?ref=tron-plugin-flake-update";
  };

  outputs = { self, nixpkgs, flake-utils, ... }@inputs:
    # it enables us to generate all outputs for each default system
    (flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = import nixpkgs {
            inherit system;
          };
          gauntlet = inputs.gauntlet.packages.${system};

          # define the wrapper script for setting up Gauntlet
          wrapperScript = pkgs.writeShellScriptBin "integration-tests" ''
            _debug() {
                level=`echo $LOG_LEVEL | tr 'A-Z' 'a-z'`
                [[ $level != "debug" ]] && return 0

                echo "time=$(date '+%Y-%m-%dT%H:%M:%S') level=DEBUG source=tron.integration-tests msg=$@"
            }

            _info() {
                level=`echo $LOG_LEVEL | tr 'A-Z' 'a-z'`
                [[ $level == "debug" ]] && return 0

                echo "time=$(date '+%Y-%m-%dT%H:%M:%S') level=INFO source=tron.integration-tests msg=$@"
            }

            ## Setup G++ environment
            _info "setting up G++ environment w/ plugins"
            # list out all files in 
            ${gauntlet.gauntlet-plugins-install}/bin/gauntlet-plugins-install &> /dev/null 2>error.log
            cat error.log

            ## Spin up G++ server with plugins
            _info "spinning up G++ environment w/ plugins"
            _debug "spinning up G++ server with plugin"
            echo "Starting G++ server with plugins:"
            ${gauntlet.default}/bin/gauntlet plugins
            ${gauntlet.default}/bin/gauntlet serve &> gauntlet.log & G_SERVER_PID=$!
            cat gauntlet.log

            _debug "G++ server liveness check"
            until nc -z localhost 8080 &> /dev/null; do 
              _debug "G++ server is not ready yet, waiting 1s"; sleep 1
            done
            

            ## Run Go Integrations test
            _info "running Go tests"
            _debug "running Go tests"
            go test -p 1 -v -count=1 -tags=integration ./...
            exit_code=$?

            ## Teardown G++ server with plugins
            _debug "teardown G++ server"
            kill $G_SERVER_PID
            wait $G_SERVER_PID

            exit $exit_code
          '';
        in
        {
          # it outputs packages all packages defined in plugins
        packages = {
          integration-tests-wrapped = wrapperScript;
        };

          # it outputs the default shell
          devShells.default =
            pkgs.mkShell {
              buildInputs =
                [
                  # add all local plugins of this project
                  self.packages.${system}.integration-tests-wrapped
                ];
            };

          # it outputs plugins as apps enabling us to `nix run '.#<app_name>'`
          apps.integration-tests = {
            type = "app";
            program = "${self.packages.${system}.integration-tests-wrapped}/bin/integration-tests";
          };
        })
    );
}