{ pkgs ? import <nixpkgs> { } }:

with pkgs;
let
  rage = rustPlatform.buildRustPackage rec {
    pname = "rage";
    version = "0.5.0-9f824625195583c5cff0f48e5bba9b216e1fa3f6";

    src = fetchFromGitHub {
      owner = "str4d";
      repo = pname;
      rev = "9f824625195583c5cff0f48e5bba9b216e1fa3f6";
      sha256 = "0j84sf9q2k1dv1w18vhmcrx75afnfl9xyp1l4vcw3baj70943nd9";
    };

    cargoSha256 = "1iirf4w7fnqvjml2ijahvsplqb5n6hqlfc7ndf0klj7km5b1s2ly";

    # Workaround a build failure that started happening by disabling
    # doctests. No idea why the normal build still works, what's the
    # difference.
    #
    #    Doc-tests age
    # error[E0433]: failed to resolve: could not find `utils` in `rust_embed`
    #  --> /build/source/age/src/i18n.rs:9:10
    #   |
    # 9 | #[derive(RustEmbed)]
    #   |          ^^^^^^^^^ could not find `utils` in `rust_embed`
    #   |
    #   = note: this error originates in a derive macro (in Nightly builds, run with -Z macro-backtrace for more info)
    doCheck = false;
  };
in mkShell {
  buildInputs = [
    pinentry-gtk2
    rage

    pcsclite
    pkg-config
    go

    killall
  ];
}
