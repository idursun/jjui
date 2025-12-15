{ inputs, ... }:
{
  systems = inputs.nixpkgs.lib.systems.flakeExposed;

  imports = [
    inputs.flake-parts.flakeModules.partitions
    ./overlays.nix
    ./partitions.nix
    ./jjui.nix
  ];
}
