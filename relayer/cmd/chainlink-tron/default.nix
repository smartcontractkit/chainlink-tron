{
  pkgs,
  rev,
}: let
  package-info = builtins.fromJSON (builtins.readFile ../../../package.json);
in
  pkgs.buildGo124Module rec {
    inherit (package-info) version;
    pname = "chainlink-tron";

    # source at the root of the module
    src = ./../..;
    subPackages = ["cmd/${pname}"];

    ldflags = [
      "-X main.Version=${package-info.version}"
      "-X main.GitCommit=${rev}"
    ];

    # pin the vendor hash (update using 'pkgs.lib.fakeHash')
    vendorHash = "sha256-d7uPn6jiFjgpK7foLEXObdfFO6l2M3R5AU6anRKvqsw=";

    # postInstall script to write version and rev to share folder
    postInstall = ''
      mkdir $out/share
      echo ${package-info.version} > $out/share/.version
      echo ${rev} > $out/share/.rev
    '';

    meta = with pkgs.lib; {
      inherit (package-info) description;
      license = licenses.mit;
      changelog = "https://github.com/smartcontractkit/${pname}/releases/tag/v${version}";
    };
  }
