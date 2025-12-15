{ lib, ... }:
{
  perSystem =
    { pkgs, ... }:
    let
      buildInputs = [ pkgs.jujutsu ]; # Runtime
      nativeBuildInputs = [ pkgs.makeWrapper ]; # Buildtime

      jjui = pkgs.buildGoModule {
        inherit buildInputs nativeBuildInputs;

        pname = "jjui";
        version = "dev";

        src = lib.fileset.toSource {
          root = ./..;
          fileset = lib.fileset.unions [
            ./../go.mod
            ./../go.sum
            ./../cmd
            ./../internal
            ./../test
          ];
        };
        vendorHash = "sha256-2TlJJY/eM6yYFOdq8CcH9l2lFHJmFrihuGwLS7jMwJ0=";
        doCheck = false;

        postInstall = ''
          wrapProgram $out/bin/jjui \
            --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.jujutsu ]}
        '';

        meta = {
          description = "A Text User Interface (TUI) designed for interacting with the Jujutsu version control system";
          homepage = "https://github.com/idursun/jjui";
          license = pkgs.lib.licenses.mit;
          maintainers = [ "idursun" ];
          platforms = pkgs.lib.platforms.unix;
          mainProgram = "jjui";
        };
      };
    in
    {
      packages = {
        default = jjui;
        inherit jjui;
      };
    };
}
