# Tokotachi (tt) Windows Release Package

This package contains the Tokotachi command-line tool (`tt`) and a convenient installer for Windows environments.

## Package Contents
- `tt.exe`: Executable binary for the Tokotachi command-line tool.
- `install.ps1`: PowerShell script for automated installation.
- `uninstall.ps1`: PowerShell script for automated uninstallation.
- `README.md`: This instruction file.

---

## Installation Instructions

### Option A: Automated Installation (Recommended)
You can run the PowerShell script to automatically place `tt.exe` in the appropriate directory and add it to your user `PATH` environment variable. No administrator privileges are required.

1. Extract the contents of this ZIP archive.
2. Open PowerShell (or Terminal) in the extracted directory.
3. Run the following command:
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\install.ps1
   ```
4. Once the installation completes successfully, **open a new terminal window** (for the path changes to take effect) and verify the installation:
   ```cmd
   tt --help
   ```

### Option B: Manual Installation
1. Move `tt.exe` to a directory of your choice (e.g., `C:\bin`).
2. Add that directory to your user `PATH` environment variable.

---

## Uninstallation Instructions
This removes all installed files and cleans up the directory from your user `PATH`.

1. Open PowerShell in the directory where `uninstall.ps1` is located.
2. Run the following command:
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\uninstall.ps1
   ```
