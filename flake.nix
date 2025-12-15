{
  description = "A Text User Interface (TUI) designed for interacting with the Jujutsu version control system";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";

    flake-compat = {
      url = "github:NixOS/flake-compat";
      flake = false;
    };
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = inputs.nixpkgs.lib.systems.flakeExposed;

      imports = [
        inputs.flake-parts.flakeModules.partitions
        ./nix/jjui.nix
      ];

      partitionedAttrs = {
        devShells = "dev";
        formatter = "dev";
        checks = "dev";
      };

      partitions.dev = {
        extraInputsFlake = ./nix/dev;
        module =
          { inputs, ... }:
          {
            imports = [
              inputs.treefmt-nix.flakeModule
              ./nix/shells.nix
              ./nix/treefmt.nix
            ];
          };
      };

      flake = {
        overlays.default = final: prev: {
          jjui = inputs.self.packages.${final.system}.jjui;
        };
      };
    };
}
