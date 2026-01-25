{ inputs, ... }:
{
  flake.overlays.default = final: _prev: {
    jjui = inputs.self.packages.${final.stdenv.hostPlatform.system}.jjui;
  };
}
