{ inputs, ... }:
{
  systems = import inputs.systems;

  flake.overlays.default = final: _prev: {
    jjui = inputs.self.packages.${final.system}.jjui;
  };

  perSystem =
    { pkgs, ... }:
    let
      jjui = pkgs.buildGoModule rec {
        name = "jjui";
        src = ./..;
        vendorHash = builtins.readFile ./vendor-hash;
        meta.mainProgram = "jjui";
      };

    in
    {
      packages.default = jjui;
      packages.jjui = jjui;
      checks.default = jjui;

      devShells.default = pkgs.mkShell {
        nativeBuildInputs = [
          pkgs.go
          pkgs.gopls
        ];
      };
    };
}
