{
  description = ''
    Flake for the cf-cli-plugin for app-autoscaler

    For more on app-autoscaler, see: <https://github.com/cloudfoundry/app-autoscaler-release>
  '';

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in {
      packages = forAllSystems (system:
        let
          nixpkgs = nixpkgsFor.${system};
        in {
          # golangci-lint fails to run, if it is compiled with a version lower than used in the project to lint:
          # https://github.com/golangci/golangci-lint/pull/4938
          # golangci-lint in NixOS is compiled by the default NixOS Go version, which is currently Go 1.22
          # To fix this, we override the golangci-lint derivation to use the Go version of the project.
          golangci-lint = nixpkgs.golangci-lint.override { buildGoModule = nixpkgs.buildGo123Module; };
      });
    };
}
