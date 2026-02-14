{
  description = "Private inputs for development purposes. These are used by the top level flake in the `dev` partition, but do not appear in consumers' lock files.";

  inputs = {
  };

  # This flake is only used for its inputs.
  outputs = { ... }: { };
}
