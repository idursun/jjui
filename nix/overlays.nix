{ inputs, ... }:
{
  flake.overlays.default = final: _prev: {
    jjui = inputs.self.packages.${final.system}.jjui;
  };
}
