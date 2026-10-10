{
  description = "httputil — HTTP utilities for Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    # go-standard (flakeModules.go-standard) composes treefmt-nix + systems
    # internally — this repo no longer declares them as direct inputs.
    go-nix-helpers = {
      url = "github:LarsArtmann/go-nix-helpers";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      ...
    }:
    let
      # benchstat is not packaged in nixpkgs; pin it from the Go module
      # proxy's source so benchmark comparisons use a fixed tool version.
      # Shared between the devShell and packages.benchstat — same arguments,
      # same derivation.
      mkBenchstat =
        pkgs:
        pkgs.buildGoModule rec {
          pname = "benchstat";
          version = "0.0.0-20260908200009-22c9c6c9d4da";

          src = pkgs.fetchFromGitHub {
            owner = "golang";
            repo = "perf";
            rev = "22c9c6c9d4da6248aedbc79f02ecedcd59f8f5f2";
            hash = "sha256-RSiI5I92l9bMWxTbHNKhcti4OKj8kD9yFxeHaWQJiFU=";
          };

          subPackages = [ "cmd/benchstat" ];

          vendorHash = "sha256-9y6O/R2fOPYAGjlIZ2lcO1TNiZPj6My3EoPRiiFZu3U=";
        };
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ inputs.go-nix-helpers.flakeModules.go-standard ];

      go-standard = {
        pname = "httputil";
        description = "HTTP utilities for Go";
        goPkgAttr = "go_1_27";
        vendorHash = "sha256-i7d92EN50fPO+afLq0blMBR6rqxKQT+3ZKG/4dOxTE0=";
        # Library repo: the package exists so the ROOT module compiles;
        # tests run via apps.test (race), not in checkPhase.
        enableCheck = false;
        enableTestCheck = false;
        enableGofumpt = true;
        enableGoimports = true;
        enableNixfmt = true;
        devShellExtraPackages = pkgs: [
          (mkBenchstat pkgs)
          pkgs.gofumpt
          pkgs.golines
          pkgs.gotools
          pkgs.trash-cli
          pkgs.d2
          pkgs.dprint
          pkgs.lychee
        ];
        devShellHook = ''
          echo "httputil dev shell — $(go version)"
          echo "  diagrams: d2 $(d2 --version 2>/dev/null || echo '?') — layout engine: elk (docs/architecture-understanding/*.d2)"
        '';
      };

      perSystem =
        {
          pkgs,
          ...
        }:
        let
          goPkg = pkgs.go_1_27;
        in
        {
          # golines on top of go-standard's enabled programs
          # (gofumpt/goimports/nixfmt).
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              goimports = {
                # goimports shells out to `go` for import resolution. In the
                # sandboxed treefmt check there is no network, so the default
                # GOTOOLCHAIN=auto fails trying to download the toolchain the
                # go 1.27.1 directive names. Wrap it: GOTOOLCHAIN=local plus
                # go_1_27 on PATH satisfies the directive without any
                # download.
                package = pkgs.writeShellApplication {
                  name = "goimports";
                  runtimeInputs = [ goPkg ];
                  text = ''
                    export GOTOOLCHAIN=local
                    export GOCACHE="''${GOCACHE:-$TMPDIR/goimports-cache}"
                    exec ${pkgs.gotools}/bin/goimports "$@"
                  '';
                };
              };
              # 120 must match .golangci.yml formatters.settings.golines
              # max-len: treefmt's default is golines' built-in 100, and a
              # diverging threshold makes `nix fmt` re-split lines that
              # `golangci-lint run` accepts (the gates fight each other).
              golines = {
                enable = true;
                maxLength = 120;
              };
            };
          };

          packages.benchstat = mkBenchstat pkgs;

          checks = {
            # The sub-module must build standalone (GOWORK=off, network off):
            # it is a separate Go module that consumers can adopt without the
            # workspace, and its zero-dependency claim is only proven when the
            # build cannot reach a module proxy.
            server-timing-standalone =
              pkgs.runCommand "server-timing-gowork-off-build"
                {
                  src = self + "/server_timing";
                  nativeBuildInputs = [ goPkg ];
                }
                ''
                  export GOWORK=off GOPROXY=off CGO_ENABLED=0 GOCACHE=$TMPDIR/go-cache
                  cd "$src"
                  go build ./...
                  go vet ./...
                  touch $out
                '';
          };

          # All bespoke apps survive verbatim; go-standard's generic
          # default/test/lint apps are mkDefault, so these win where names
          # collide.
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

            vulncheck = {
              type = "app";
              meta.description = "Run govulncheck on all packages";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-vulncheck";
                    runtimeInputs = [
                      goPkg
                      pkgs.govulncheck
                    ];
                    text = ''
                      export GOWORK=off
                      exec ${pkgs.govulncheck}/bin/govulncheck ./...
                    '';
                  };
                in
                "${script}/bin/run-vulncheck";
            };

            bench = {
              type = "app";
              meta.description = "Run the documented benchmark baseline (3s×5 protocol, benchstat-ready)";
              program =
                let
                  script = pkgs.writeShellApplication {
                    name = "run-bench";
                    runtimeInputs = [ goPkg ];
                    text = ''
                      export GOWORK=off
                      exec ${goPkg}/bin/go test -run='^$' -bench . -benchtime=3s -count=5 "$@"
                    '';
                  };
                in
                "${script}/bin/run-bench";
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
