{
  perSystem =
    { pkgs, ... }:
    {
      treefmt = {
        programs.nixfmt = {
          enable = pkgs.lib.meta.availableOn pkgs.stdenv.buildPlatform pkgs.nixfmt-rfc-style.compiler;
          package = pkgs.nixfmt-rfc-style;
        };
        programs.gofmt.enable = true;
      };
    };
}
