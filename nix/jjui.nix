{ inputs, ... }:
{
  perSystem =
    { pkgs, ... }:
    let
      jjui = pkgs.callPackage ./package.nix {
        version = inputs.self.shortRev or inputs.self.dirtyShortRev or "dev";
      };
    in
    {
      packages = {
        inherit jjui;
        default = jjui;
      };

      checks = {
        inherit jjui;
      };
    };
}
