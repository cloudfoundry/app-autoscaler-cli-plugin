{ stdenv }:
stdenv.buildGoModule {
  pname = "app-autoscaler-cli-plugin";
  version = "latest";
  src = ./.;
}