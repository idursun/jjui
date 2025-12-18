{
  perSystem =
    { pkgs, ... }:
    {
      treefmt = {
        programs.nixfmt.enable = true;
        programs.gofmt.enable = true;
        programs.yamlfmt.enable = true;
        programs.taplo.enable = true;
        programs.prettier.enable = true;

        settings.formatter.taplo.excludes = [ "internal/config/default/config.toml" ];
        settings.formatter.prettier.includes = [
          "*.md"
          "*.mdx"
        ];
      };
    };
}
