{ inputs, ... }:
{
  systems = import inputs.systems;

  flake.overlays.default = final: _prev: {
    jjui = inputs.self.packages.${final.system}.jjui;
  };

  perSystem =
    { pkgs, lib, ... }:
    let
      self = inputs.self;
      version = if (self ? rev) then self.rev else "dirty-${self.dirtyRev}";
      jjui = pkgs.buildGoModule {
        name = "jjui";
        src = lib.cleanSource ./..;

        ldflags = [ "-X main.Version=${version}" ];
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
