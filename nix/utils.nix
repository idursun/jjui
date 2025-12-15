{
  perSystem =
    { pkgs, ... }:
    let
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
      apps = {
        update-vendor-hash = {
          type = "app";
          program = "${update-vendor-hash}/bin/update-vendor-hash";
        };
      };
    };
}
