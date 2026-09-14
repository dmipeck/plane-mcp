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
          inherit (pkgs) lib;
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

          plane-mcp = pkgs.buildGoModule {
            pname = "plane-mcp";
            version = "0.1.0";
            inherit src;
            vendorHash = "sha256-gKpXmBuFTVOoKZJ3wj1SzS261Fvp/r12u6FuDeCkcY4=";
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

          # Linux-only OCI image (stdio MCP server + CA bundle for Plane HTTPS).
          docker = pkgs.dockerTools.buildLayeredImage {
            name = "plane-mcp";
            tag = "latest";
            contents = [
              plane-mcp
              pkgs.dockerTools.caCertificates
            ];
            config = {
              Entrypoint = [ "/bin/plane-mcp" ];
              # MCP hosts attach over stdio; keep the process in the foreground.
              Cmd = [ ];
            };
          };
        in
        {
          packages = {
            inherit plane-mcp;
            default = plane-mcp;
          }
          // lib.optionalAttrs pkgs.stdenv.hostPlatform.isLinux {
            inherit docker;
          };

          apps.plane-mcp = {
            type = "app";
            program = "${self'.packages.plane-mcp}/bin/plane-mcp";
          };
          apps.default = self'.apps.plane-mcp;

          pre-commit = {
            check.enable = false;
            settings.hooks = {
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

          checks = {
            plane-mcp = self'.packages.plane-mcp;
          }
          // lib.optionalAttrs pkgs.stdenv.hostPlatform.isLinux {
            docker = self'.packages.docker;
          };
        };
    };
}
