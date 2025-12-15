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
        vendorHash = lib.strings.trim (builtins.readFile ./vendor-hash);
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

      update-vendor-hash = pkgs.writeShellApplication {
        name = "update-vendor-hash";

        runtimeInputs = with pkgs; [
          gnugrep
          gnused
        ];

        text = ''
          HASH_FILE="nix/vendor-hash"

          if BUILD_OUTPUT=$(nix build .#jjui --no-link 2>&1); then
            echo "vendor-hash is up to date"
            exit 0
          fi

          NEW_HASH=$(echo "$BUILD_OUTPUT" | grep -E '^\s+got:' | sed -E 's/.*got:\s+//' | head -1)

          if [[ -z "$NEW_HASH" ]]; then
            echo "Build failed without hash mismatch:"
            echo "$BUILD_OUTPUT"
            exit 1
          fi

          echo "$NEW_HASH" > "$HASH_FILE"
          echo "Updated $HASH_FILE to $NEW_HASH"
        '';
      };
    in
    {
      packages = {
        default = jjui;
        inherit jjui update-vendor-hash;
      };
    };
}
