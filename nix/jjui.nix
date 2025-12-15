{ lib, ... }:
{
  perSystem =
    { pkgs, ... }:
    {
      packages = {
        default = pkgs.callPackage ./package.nix { };
        jjui = pkgs.callPackage ./package.nix { };
      };
    };
}
