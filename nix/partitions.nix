{ inputs, ... }:
{
  imports = [
    inputs.flake-parts.flakeModules.partitions
  ];

  partitionedAttrs = {
    devShells = "dev";
    apps = "dev";
  };

  partitions.dev = {
    extraInputsFlake = ./dev;
    module =
      { inputs, ... }:
      {
        imports = [
          ./shells.nix
          ./utils.nix
        ];
      };
  };
}
