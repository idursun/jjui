{
  lib,
  buildGoModule,
}:

buildGoModule {
  pname = "jjui";
  version = "dev"; # TODO: update this to use version/git hash

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

  meta = {
    description = "A Text User Interface (TUI) designed for interacting with the Jujutsu version control system";
    homepage = "https://github.com/idursun/jjui";
    license = lib.licenses.mit;
    maintainers = [ "idursun" ];
    platforms = lib.platforms.unix;
    mainProgram = "jjui";
  };
}
