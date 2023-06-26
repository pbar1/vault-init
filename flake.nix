{
  inputs = {
    naersk.url = "github:nix-community/naersk/master";
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils, naersk }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        naersk-lib = pkgs.callPackage naersk { };
        bin = naersk-lib.buildPackage ./.;
      in
      {
        devShell = with pkgs; mkShell {
          buildInputs = [ cargo rustc rustfmt pre-commit rustPackages.clippy ];
          RUST_SRC_PATH = rustPlatform.rustLibSrc;
        };
        defaultPackage = bin;
        packages.dockerImage = pkgs.dockerTools.streamLayeredImage {
          name = "ghcr.io/pbar1/vault-init";
          tag = "latest";
          contents = [ bin ];
          config = {
            Cmd = [ "${bin}/bin/vault-init" ];
            Labels = {
              "org.opencontainers.image.source" = "https://github.com/pbar1/vault-init";
            };
          };
        };
      });
}
