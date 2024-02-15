{ stdenv }:
stdenv.mkDerivation {
  pname = "hello";
  version = "2.12.1";

  src = ./.;
}