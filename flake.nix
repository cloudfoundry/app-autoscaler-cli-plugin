{
  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/nixos-unstable;
  };

  outputs = { self, nixpkgs }: {
        hello = nixpkgs.callPackage ./app-autoscaler-cli-plugin.nix {};
    };
}