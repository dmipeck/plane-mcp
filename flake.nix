{
  description = "Multi-workspace Plane MCP server (Go)";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    git-hooks.url = "github:cachix/git-hooks.nix";
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.git-hooks.flakeModule
      ];

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      perSystem =
        {
          config,
          pkgs,
          self',
          ...
        }:
        let
          src = pkgs.lib.cleanSourceWith {
            src = self;
            filter =
              path: _type:
              let
                base = baseNameOf path;
              in
              base != ".git"
              && base != "result"
              && base != ".agent"
              && base != ".direnv";
          };
        in
        {
          packages.plane-mcp = pkgs.buildGoModule {
            pname = "plane-mcp";
            version = "0.1.0";
            inherit src;
            vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
            subPackages = [ "." ];
            ldflags = [
              "-s"
              "-w"
            ];
            meta = {
              description = "Multi-workspace Plane MCP server over stdio";
              mainProgram = "plane-mcp";
            };
          };

          packages.default = self'.packages.plane-mcp;

          apps.plane-mcp = {
            type = "app";
            program = "${self'.packages.plane-mcp}/bin/plane-mcp";
          };
          apps.default = self'.apps.plane-mcp;

          pre-commit.settings.hooks = {
            gofmt.enable = true;
            govet = {
              enable = true;
              name = "go vet";
              entry = "${pkgs.go}/bin/go vet ./...";
              files = "\\.go$";
              pass_filenames = false;
            };
            golangci-lint = {
              enable = true;
              package = pkgs.golangci-lint;
            };
          };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              golangci-lint
              govulncheck
              git
            ];
            shellHook = ''
              ${config.pre-commit.installationScript}
            '';
          };

          checks.plane-mcp = self'.packages.plane-mcp;
        };
    };
}
