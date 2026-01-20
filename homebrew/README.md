# Homebrew Tap Setup

This directory contains the Homebrew formula for MissionControl.

## Setting Up Your Tap

1. Create a new GitHub repository named `homebrew-tap`:
   ```bash
   # On GitHub, create: DarlingtonDeveloper/homebrew-tap
   ```

2. Clone and add the formula:
   ```bash
   git clone https://github.com/DarlingtonDeveloper/homebrew-tap.git
   cd homebrew-tap
   mkdir Formula
   cp /path/to/MissionControl/homebrew/mission-control.rb Formula/
   git add .
   git commit -m "Add mission-control formula"
   git push
   ```

3. Create a release with binaries:
   ```bash
   cd /path/to/MissionControl
   make release
   # Upload dist/*.tar.gz to GitHub Releases
   ```

4. Update formula SHA256 hashes:
   ```bash
   shasum -a 256 dist/*.tar.gz
   # Update the sha256 lines in Formula/mission-control.rb
   ```

## Installing via Homebrew

Once the tap is set up:

```bash
brew tap DarlingtonDeveloper/tap
brew install mission-control
```

## Testing Locally

```bash
brew install --build-from-source ./homebrew/mission-control.rb
```
