{
  description = "A Text User Interface (TUI) designed for interacting with the Jujutsu version control system";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
    flake-parts.url = "github:hercules-ci/flake-parts";
    flake-compat.url = "https://flakehub.com/f/edolstra/flake-compat/1.tar.gz";
    treefmt-nix.url = "github:numtide/treefmt-nix";
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      systems,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import systems;
      imports = [
        inputs.treefmt-nix.flakeModule
      ];

      perSystem =
        {
          self',
          pkgs,
          ...
        }:
        let
          buildInputs = [ pkgs.jujutsu ]; # Runtime
          nativeBuildInputs = [ pkgs.makeWrapper ]; # Buildtime

          jjui = pkgs.buildGoModule {
            inherit buildInputs nativeBuildInputs;
            pname = "jjui";
            version = "dev";

            src = ./.;
            vendorHash = "sha256-2TlJJY/eM6yYFOdq8CcH9l2lFHJmFrihuGwLS7jMwJ0=";

            doCheck = false;

            postInstall = ''
              wrapProgram $out/bin/jjui \
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.jujutsu ]}
            '';

            meta = {
              description = "A Text User Interface (TUI) designed for interacting with the Jujutsu version control system";
              homepage = "https://github.com/idursun/jjui";
              license = pkgs.lib.licenses.mit;
              maintainers = [ "idursun" ];
              platforms = pkgs.lib.platforms.unix;
              mainProgram = "jjui";
            };
          };
        in
        {
          packages = {
            default = jjui;
            inherit jjui;
          };

          devShells.default = pkgs.mkShell {
            name = "jjui-dev";

            buildInputs =
              with pkgs;
              [
                # Go toolchain
                go
                gotools
              ]
              ++ buildInputs;

            # Environment variables for development
            CGO_ENABLED = "0";
          };

          treefmt = {
            programs.nixfmt.enable = pkgs.lib.meta.availableOn pkgs.stdenv.buildPlatform pkgs.nixfmt-rfc-style.compiler;
            programs.nixfmt.package = pkgs.nixfmt-rfc-style;
            programs.gofmt.enable = true;
          };
        };

      flake = {
        overlays.default = final: prev: {
          jjui = inputs.self.packages.${final.system}.jjui;
        };
      };
    };
}
