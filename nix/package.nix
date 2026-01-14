{
  lib,
  buildGoModule,
  version ? "dev",
}:

buildGoModule {
  inherit version;
  pname = "jjui";

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
  doCheck = true;

  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
  ];

  meta = {
    description = "A Text User Interface (TUI) designed for interacting with the Jujutsu version control system";
    homepage = "https://github.com/idursun/jjui";
    license = lib.licenses.mit;
    maintainers =
      with lib.maintainers;
      [
        adda
        doprz
      ]
      ++ [
        "idursun"
        "vic"
      ];
    platforms = lib.platforms.unix;
    mainProgram = "jjui";
  };
}
