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
          # You can set JJUI_CONF_DIR to a gitignored directory inside your
          # working copy to test configuration changes without affecting your
          # system-wide jjui config.
          #
          # Example (uncomment or add to your shell):
          #   export JJUI_CONF_DIR="$PWD/.dev-config"
          #   mkdir -p "$JJUI_CONF_DIR"
          #
          # This allows you to experiment with jjui configurations during
          # development without modifying your personal settings.
        '';
      };
    };
}
