{
  description = ''
    Flake for the cf-cli-plugin for app-autoscaler

    For more on app-autoscaler, see: <https://github.com/cloudfoundry/app-autoscaler-release>
  '';

  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/nixos-23.11;
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in flake-utils.lib.eachSystem supportedSystems (system: {
      packages = {
        app-autoscaler-cli-plugin = nixpkgsFor.${system}.buildGoModule rec {
          pname = "app-autoscaler-cli-plugin";
          version =
            let
              lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";
            in "${builtins.substring 0 8 lastModifiedDate}-dev";
          src = ./.;
          vendorHash = "sha256-NzEStcOv8ZQsHOA8abLABKy+ZE3/SiYbRD/ZVxo0CEk=";

          doCheck = false;

          meta = {
            description = ''
              App-AutoScaler plug-in provides the command line interface to manage
              [App AutoScaler](<https://github.com/cloudfoundry-incubator/app-autoscaler>)
              policies, retrieve metrics and scaling event history.
            '';
            homepage = "https://github.com/cloudfoundry/app-autoscaler-cli-plugin";
            license = [nixpkgsLib.licenses.apsl20];
          };
        };
      };

      devShells =
        let
          nixpkgs = nixpkgsFor.${system};
        in {
          default = nixpkgs.mkShell {
            buildInputs = with nixpkgs; [
              delve
              go
              gopls
            ];
          };
        };
    });
}