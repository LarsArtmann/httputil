{
  description = "httputil — HTTP utilities for Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    systems.url = "github:nix-systems/default";

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          system,
          ...
        }:
        let
          goPkg = pkgs.go_1_26;

          src = lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              (lib.fileset.fileFilter (file: lib.hasSuffix ".go" file.name) ./.)
              ./go.mod
              ./go.sum
            ];
          };
        in
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              gofumpt.enable = true;
              goimports.enable = true;
              golines.enable = true;
              nixfmt.enable = true;
            };
          };

          devShells.default = pkgs.mkShellNoCC {
            packages = [
              goPkg
              pkgs.golangci-lint
              pkgs.gofumpt
              pkgs.golines
              pkgs.gotools
              pkgs.trash-cli
            ];

            shellHook = ''
              echo "httputil dev shell — $(go version)"
            '';
          };

          checks = {
            format = config.treefmt.build.check self;
          };

          apps = {
            test = {
              type = "app";
              meta.description = "Run Go tests with race detection";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-tests";
                    runtimeInputs = [ goPkg ];
                    text = ''
                      export GOWORK=off
                      exec ${goPkg}/bin/go test ./... -race -count=1 "$@"
                    '';
                  };
                in
                "${script}/bin/run-tests";
            };

            test-race = {
              type = "app";
              meta.description = "Run Go tests with race detection";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-tests-race";
                    runtimeInputs = [ goPkg ];
                    text = ''
                      export GOWORK=off
                      exec ${goPkg}/bin/go test ./... -race -count=1 "$@"
                    '';
                  };
                in
                "${script}/bin/run-tests-race";
            };

            build = {
              type = "app";
              meta.description = "Build all Go packages";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-build";
                    runtimeInputs = [ goPkg ];
                    text = ''
                      export GOWORK=off
                      exec ${goPkg}/bin/go build ./...
                    '';
                  };
                in
                "${script}/bin/run-build";
            };

            vet = {
              type = "app";
              meta.description = "Run go vet on all packages";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-vet";
                    runtimeInputs = [ goPkg ];
                    text = ''
                      export GOWORK=off
                      exec ${goPkg}/bin/go vet ./...
                    '';
                  };
                in
                "${script}/bin/run-vet";
            };

            lint = {
              type = "app";
              meta.description = "Run golangci-lint on all packages";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-lint";
                    runtimeInputs = [
                      goPkg
                      pkgs.golangci-lint
                    ];
                    text = ''
                      export GOWORK=off
                      exec ${pkgs.golangci-lint}/bin/golangci-lint run ./...
                    '';
                  };
                in
                "${script}/bin/run-lint";
            };

            coverage = {
              type = "app";
              meta.description = "Run Go tests with coverage report";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-coverage";
                    runtimeInputs = [ goPkg ];
                    text = ''
                      export GOWORK=off
                      ${goPkg}/bin/go test ./... -coverprofile=coverage.out -covermode=atomic "$@"
                      ${goPkg}/bin/go tool cover -func=coverage.out
                    '';
                  };
                in
                "${script}/bin/run-coverage";
            };

            clean = {
              type = "app";
              meta.description = "Clean test cache and coverage files";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-clean";
                    runtimeInputs = [
                      goPkg
                      pkgs.trash-cli
                    ];
                    text = ''
                      ${pkgs.trash-cli}/bin/trash-put coverage.out 2>/dev/null || true
                      ${goPkg}/bin/go clean -testcache
                    '';
                  };
                in
                "${script}/bin/run-clean";
            };
          };
        };
    };
}
