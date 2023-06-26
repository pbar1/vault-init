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
        packages.dockerImage = pkgs.dockerTools.buildLayeredImage {
          name = "vault-init";
          tag = "beta";
          contents = [ bin ];
          config = {
            Cmd = [ "${bin}/bin/vault-init" ];
          };
        };
        defaultPackage = bin;
      });
}
