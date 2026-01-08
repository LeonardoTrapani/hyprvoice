{
  description = "Voice-powered typing for Hyprland/Wayland";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs, ... }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "hyprvoice";
            version = self.shortRev or "dirty";
            src = ./.;

            vendorHash = "sha256-qYZGccprn+pRbpVeO1qzSOb8yz/j/jdzPMxFyIB9BNA=";

            subPackages = [ "cmd/hyprvoice" ];
          };
        }
      );
    };
}
