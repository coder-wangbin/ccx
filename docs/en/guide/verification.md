# Verifying Release Artifacts

CCX releases are signed using [Sigstore](https://www.sigstore.dev/) keyless signing on the checksums manifest. Verifying the signature proves that the release artifacts were built by the `BenedictKing/ccx` GitHub Actions workflow and have not been tampered with.

Windows GitHub installers are also Authenticode-signed through the SignPath Foundation. Sigstore verifies the Release manifest and CI origin; Authenticode lets Windows verify installer publisher and file integrity.

## How It Works

1. During CI, GitHub Actions requests a **short-lived signing certificate** (valid ~10 minutes) from Sigstore's Fulcio CA via the OIDC protocol. The certificate binds the signing identity to the GitHub Actions workflow.
2. `cosign` signs the `checksums.txt` file with that certificate, producing a Sigstore bundle (`.sigstore.json`).
3. The signature is recorded in the Rekor public transparency log, which anyone can audit.
4. When verifying, cosign extracts the certificate from the bundle and checks that the signature matches the expected GitHub Actions identity.

The entire process requires **no key management** from the project maintainers.

## Installing cosign

macOS:

```bash
brew install cosign
```

Linux (binary):

```bash
# See https://docs.sigstore.dev/cosign/installation/
COSIGN_VERSION="v3.8.0"
curl -LO "https://github.com/sigstore/cosign/releases/download/${COSIGN_VERSION}/cosign-linux-amd64"
chmod +x cosign-linux-amd64
sudo mv cosign-linux-amd64 /usr/local/bin/cosign
```

Windows:

```powershell
choco install cosign
```

## Verification Steps

### 1. Download checksums and signature files

Download from [GitHub Releases](https://github.com/BenedictKing/ccx/releases):

- `checksums.txt` — cross-platform SHA256 manifest
- `checksums.txt.sigstore.json` — the Sigstore signature bundle

For single-platform verification, use `checksums-{platform}.txt` and `checksums-{platform}.txt.sigstore.json`:

| Platform | Manifest | Signature Bundle |
|----------|----------|-----------------|
| macOS | `checksums-macos.txt` | `checksums-macos.txt.sigstore.json` |
| Windows | `checksums-windows.txt` | `checksums-windows.txt.sigstore.json` |
| Linux | `checksums-linux.txt` | `checksums-linux.txt.sigstore.json` |

### 2. Verify the checksums manifest signature

```bash
VERSION=v2.7.12  # Replace with the target version

cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/BenedictKing/ccx/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt
```

A successful verification prints `Verified checksums.txt` and returns exit code 0.

If you don't know the exact version, use a regex pattern:

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp "^https://github\.com/BenedictKing/ccx/\.github/workflows/release\.yml@refs/tags/v.*$" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt
```

### 3. Verify individual file SHA256

After signature verification passes, check the downloaded files against the manifest:

Linux:

```bash
sha256sum -c checksums.txt --ignore-missing
```

macOS (some `shasum` versions don't support `--ignore-missing`, compare manually):

```bash
shasum -a 256 ccx-darwin-arm64
# Compare the output with the corresponding line in checksums.txt
```

### 4. Verify Windows Authenticode signature

You can verify SignPath code signing for Windows GitHub installers with PowerShell:

```powershell
$version = "2.7.12"  # Replace with the target version without v
$signature = Get-AuthenticodeSignature ".\CCX-Desktop-$version-windows-amd64-setup.exe"
$signature.Status
$signature.SignerCertificate.Subject
```

`Status` should be `Valid`. If it shows `NotSigned`, `UnknownError`, or unexpected certificate details, do not run the installer. Re-download it from the official Release and verify SHA256 again.

## Common Verification Failures

| Symptom | Cause |
|---------|-------|
| `error: verifying bundle: mismatched identities` | Signature was not produced by this project's CI, or wrong version specified |
| `error: matching certificate identity failed` | `--certificate-identity` doesn't match the actual signing identity; check the version number |
| `error: tlog entry is not trusted` | Rekor transparency log verification failed; the bundle file may be corrupted |
| `error: no matching signatures` | `checksums.txt` content differs from when it was signed; file may have been corrupted during download |
| Signature passes but SHA256 mismatch | The downloaded binary itself is corrupted; re-download and try again |

## Artifact Descriptions

| File | Contents |
|------|----------|
| `checksums.txt` | SHA256 hashes for all cross-platform artifacts (`sha256sum` format) |
| `checksums.txt.sigstore.json` | Sigstore bundle: signature, OIDC certificate, Rekor transparency log entry |
| `checksums-{platform}.txt` | Single-platform SHA256 manifest |
| `checksums-{platform}.txt.sigstore.json` | Single-platform Sigstore bundle |
| `{artifact}.sha256` | Individual artifact SHA256 (hex hash only, used by updater for automated verification) |
| `CCX-Desktop-*-windows-*-setup.exe` | SignPath Authenticode-signed Windows GitHub installer |
| `CCX-Desktop-*-windows-*-store.msix` | Windows Store/MSIX submission and validation package; public installs should prefer Microsoft Store |

## Learn More

- [Sigstore Website](https://www.sigstore.dev/)
- [cosign Documentation](https://docs.sigstore.dev/cosign/)
- [Rekor Transparency Log](https://docs.sigstore.dev/logging/overview/)
- [SLSA Supply Chain Security](https://slsa.dev/)
