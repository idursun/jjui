{
  perSystem =
    { pkgs, ... }:
    {
      devShells.default = pkgs.mkShell {
        name = "jjui-dev";
        buildInputs = with pkgs; [
          # Go toolchain
          go
          gotools

          jujutsu
        ];

        # Environment variables for development
        CGO_ENABLED = "0";

        shellHook = ''
          if [[ -z "''${JJUI_CONF_DIR:-}" ]]; then
            echo "Pro Tip: export JJUI_CONF_DIR=\"\$PWD/.dev-config\" to use a local config directory"
          fi
        '';
      };
    };
}
