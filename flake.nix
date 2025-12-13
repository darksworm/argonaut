{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      flake-utils,
      nixpkgs,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        env.CGO_ENABLED = 0;
        packages.default = pkgs.buildGoModule {
          pname = "argonaut";
          name = "argonaut";
          meta = with pkgs.lib; {
            description = "Keyboard-first terminal UI for Argo CD";
            homepage = "https://github.com/darksworm/argonaut";
            license = licenses.gpl3Only;
            maintainers = [
              maintainers.okwilkins
              "darksworm"
            ];
            mainProgram = "argonaut";
          };
          src = ./.;
          vendorHash = "sha256-xln/WmZbi0+rHqMMHRgt0ar/EaBDNscCsd/NckJZnMw=";
          proxyVendor = true;
          subPackages = [ "cmd/app" ];
          ldflags = [
            "-s"
            "-w"
            "-X main.commit=${self.rev or "dev"}"
            "-X main.buildDate=1970-01-01T00:00:00Z"
          ];

          # Skip tests as Nix has limited network access
          checkFlags = [
            "-skip=TestLoadPool|TestNewHTTP"
          ];
          postInstall = ''
            mv $out/bin/app $out/bin/argonaut
          '';
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/argonaut";
        };
        devShells.default = pkgs.mkShell {
          inputsFrom = [ self.packages.${system}.default ];
          packages = with pkgs; [
            self.packages.${system}.default
            argocd
            delta
            go_1_24
            gopls
            gotools
            golangci-lint
          ];
          shellHook = ''
            echo "Argonaut development environment"
            echo "Go version: $(go version)"
          '';
        };
        checks = {
          build = self.packages.${system}.default;
          format = pkgs.runCommand "check-format" { } ''
            ${pkgs.nixpkgs-fmt}/bin/nixpkgs-fmt --check ${./.}
            touch $out
          '';
        };
      }
    );
}
