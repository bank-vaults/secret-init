# `secret-init`

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/bank-vaults/secret-init/ci.yaml?branch=main&style=flat-square)](https://github.com/bank-vaults/secret-init/actions/workflows/ci.yaml?query=workflow%3ACI)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/bank-vaults/secret-init/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/bank-vaults/secret-init)

**Minimalistic init system for containers injecting secrets from various secret stores.**

## Usage

TODO

## Development

**For an optimal developer experience, it is recommended to install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html).**

_Alternatively, install [Go](https://go.dev/dl/) on your computer then run `make deps` to install the rest of the dependencies._

Make sure Docker is installed with Compose and Buildx.

Run project dependencies:

```shell
make up
```

Build a binary:

```shell
make build
```

Run the test suite:

```shell
make test
```

Run linters:

```shell
make lint # pass -j option to run them in parallel
```

Some linter violations can automatically be fixed:

```shell
make fmt
```

Build artifacts locally:

```shell
make artifacts
```

Once you are done either stop or tear down dependencies:

```shell
make stop

# OR

make down
```

## License

The project is licensed under the [Apache 2.0 License](LICENSE).
