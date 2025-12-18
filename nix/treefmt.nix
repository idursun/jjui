{
  perSystem =
    { pkgs, ... }:
    let
      tomlPath = ../treefmt.toml;
      tomlConfig =
        if builtins.pathExists tomlPath then builtins.fromTOML (builtins.readFile tomlPath) else { };
    in
    {
      treefmt.settings.formatter = tomlConfig.formatter or { };
    };
}
