# Debian installation

Leamas publishes native Linux amd64 Debian packages through GitHub Releases.
The stable package name is `leamas_<version>_amd64.deb`; the corresponding
release asset `SHA256SUMS` contains the package checksum.

## GitHub CLI installation

```bash
gh release download v0.1.0 \
  --repo s1onique/leamas \
  --pattern 'leamas_0.1.0_amd64.deb' \
  --pattern SHA256SUMS

sha256sum --check SHA256SUMS
sudo apt install ./leamas_0.1.0_amd64.deb

leamas version
```

The checksum command must run from the directory containing both downloaded
assets. It verifies the package before installation.

## Browser or curl installation

Open the [Leamas v0.1.0 release page](https://github.com/s1onique/leamas/releases/tag/v0.1.0)
and download both `leamas_0.1.0_amd64.deb` and `SHA256SUMS`. From the directory
containing the two files, verify the checksum and install the local package:

```bash
sha256sum --check SHA256SUMS
sudo apt install ./leamas_0.1.0_amd64.deb
```

For scripted downloads, use the release asset URL with `curl -fL -o` and then
run the same checksum command. Do not pipe downloaded content directly into a
privileged shell.

The naming convention for future releases remains
`leamas_<new-version>_amd64.deb`, with a matching `SHA256SUMS` asset.

## Removal

```bash
sudo apt remove leamas
```

## Upgrade

GitHub Releases are not an APT repository and do not provide automatic APT
repository updates. Download and verify the new package, then install it as a
local Debian file:

```bash
sudo apt install ./leamas_<new-version>_amd64.deb
```
