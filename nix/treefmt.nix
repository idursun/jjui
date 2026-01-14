{
  perSystem =
    { lib, ... }:
    let
      tomlPath = ../treefmt.toml;
      tomlConfig =
        if builtins.pathExists tomlPath then builtins.fromTOML (builtins.readFile tomlPath) else { };

      # Remove "command" from each formatter since treefmt.programs.* handles that
      filterCommands = lib.mapAttrs (_name: formatter: builtins.removeAttrs formatter [ "command" ]);
    in
    {
      treefmt = {
        programs.nixfmt.enable = true;
        programs.gofmt.enable = true;
        programs.yamlfmt.enable = true;
        programs.taplo.enable = true;

        settings.formatter = filterCommands (tomlConfig.formatter or { });
      };
    };
}
