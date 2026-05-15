# WinGet manifests

Hand-curated manifests for the Microsoft [WinGet](https://github.com/microsoft/winget-cli) package manager.

## Layout

The directory tree follows the convention required by
[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs):

```
manifests/h/heidihowilson/skillscope/<version>/
  heidihowilson.skillscope.yaml              # version
  heidihowilson.skillscope.installer.yaml    # installer (url, sha256, arch)
  heidihowilson.skillscope.locale.en-US.yaml # locale (name, description, tags)
```

## Submitting a new version

Until the goreleaser `winget:` block is wired up (it's commented in
`.goreleaser.yaml`, pending a `WINGET_PKGS_GITHUB_TOKEN` PAT and a
fork of `microsoft/winget-pkgs`), every new version's manifests need
to be submitted by hand:

1. Copy the previous version's directory to a new one named after the
   new tag.
2. Bump `PackageVersion` in all three files.
3. Update `InstallerUrl` and `InstallerSha256` in the installer
   manifest. The sha256 comes from the release's `checksums.txt`:

   ```sh
   curl -sSL https://github.com/heidihowilson/skillscope/releases/download/vX.Y.Z/checksums.txt \
     | grep windows_x86_64 \
     | awk '{print toupper($1)}'
   ```
4. Update `ReleaseNotesUrl` in the locale manifest.
5. Open a PR against `microsoft/winget-pkgs` with the new directory.

## Validation

`winget validate --manifest manifests/h/heidihowilson/skillscope/<version>/` from
a Windows machine with the WinGet CLI installed.
