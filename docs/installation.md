# Installation

`repomap` is a single, statically-linked binary with no runtime dependencies. It
needs a `git` binary on `PATH` only for the `impact` command.

## Homebrew (macOS / Linux)

```bash
brew install karanjasani/tap/repomap
```

Upgrade with `brew upgrade repomap`.

## `go install`

Requires Go 1.24 or newer.

```bash
go install github.com/karanjasani/agentkit/cmd/repomap@latest
```

The binary is placed in `$(go env GOBIN)` (or `$(go env GOPATH)/bin`). Make sure
that directory is on your `PATH`. Builds installed this way still report an
accurate `tool_version` because the version is recovered from
`runtime/debug.ReadBuildInfo()` when linker flags are absent.

## Linux packages (`.deb` / `.rpm` / `.apk`)

Download the package for your architecture from the
[releases page](https://github.com/karanjasani/agentkit/releases) and install it:

```bash
# Debian / Ubuntu
sudo dpkg -i repomap_*_linux_amd64.deb

# Fedora / RHEL
sudo rpm -i repomap_*_linux_amd64.rpm

# Alpine
sudo apk add --allow-untrusted repomap_*_linux_amd64.apk
```

## Prebuilt archives

Every release publishes cross-platform archives (`darwin`, `linux`, `windows`;
`amd64` and `arm64`), a `checksums.txt`, an SBOM, and a keyless
[cosign](https://docs.sigstore.dev/) signature.

Verify a download before use:

```bash
# Verify the checksum
sha256sum -c checksums.txt --ignore-missing

# Verify the cosign signature (keyless / Sigstore)
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity-regexp 'https://github.com/karanjasani/agentkit' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

Then extract and place `repomap` on your `PATH`:

```bash
tar -xzf repomap_*_linux_amd64.tar.gz
sudo mv repomap /usr/local/bin/
```

## Verify the install

```bash
repomap --version
repomap overview --dir /path/to/a/go/module | head
```
