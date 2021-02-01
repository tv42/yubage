{ pkgs ? import <nixpkgs> {} }:

with pkgs;
mkShell {
  buildInputs = [
    pcsclite
    pkg-config
    go
    gopls
    go-tools
    goimports
    go-outline
    nixpkgs-fmt
  ];
}
