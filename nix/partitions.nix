{ inputs, ... }:
{
  imports = [
    inputs.flake-parts.flakeModules.partitions
  ];

  partitionedAttrs = {
    devShells = "dev";
    formatter = "dev";
    checks = "dev";
    apps = "dev";
  };

  partitions.dev = {
    extraInputsFlake = ./dev;
    module =
      { inputs, ... }:
      {
        imports = [
          inputs.treefmt-nix.flakeModule
          ./shells.nix
          ./treefmt.nix
          ./utils.nix
        ];
      };
  };
}
