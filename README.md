# imprint

A small, intuitive app to flash ISOs and disk images to external drives e.g. USB drives.

![Screenshot of Imprint](https://f002.backblazeb2.com/file/retrixe-storage-public/imprint/imprint.png)

## Quick Start

1. Download the latest version of Imprint from the [releases page](https://github.com/retrixe/imprint/releases).
2. Run Imprint:
   - ~~On Windows, double-click the `imprint.exe` file.~~\
     Windows is currently not supported. Support is being tracked in [issue #3](https://github.com/retrixe/imprint/issues/3).
   - On macOS, extract the archive and double-click the `imprint` app.
   - On Linux, extract the archive and double-click the `imprint` app.
3. Proceed to flash your drive by following the steps in the app 🎉

## Supported Images / Hardware

This app is tested with a variety of ISOs from various Linux distributions e.g. Ubuntu, Fedora, openSUSE, Raspbian, etc. It should work with all disk images which can be flashed directly through `dd`.

Hardware regularly tested against include SD cards, USB flash drives, and external USB hard drives.

⚠️ Support for CD/DVD drives is untested. Flashing to a CD/DVD using this tool may result in a non-functional boot media. If you would like to hack on this, please open an issue.

⚠️ Windows ISOs are currently not supported, since they require special handling. Support is being tracked in [issue #2](https://github.com/retrixe/imprint/issues/2).
