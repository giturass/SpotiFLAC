# SpotiFLAC

Download Spotify tracks in FLAC quality from Tidal, Qobuz & Amazon Music.

![Android Build](https://github.com/zarzet/SpotiFLAC-Android/actions/workflows/android-build.yml/badge.svg)
![iOS Build](https://github.com/zarzet/SpotiFLAC-Android/actions/workflows/ios-build.yml/badge.svg)

## Features

- 🔍 Search Spotify tracks, albums, and playlists
- 📥 Download in FLAC quality from multiple sources (Tidal, Qobuz, Amazon Music)
- 🔄 Automatic fallback to available services
- 🎵 Embedded metadata and cover art
- 📝 Lyrics support (synced and plain)
- 🎨 Material 3 Expressive UI with dynamic colors
- 📱 Cross-platform: Android & iOS

## Download

### Latest Release
Download the latest version from [Releases](https://github.com/zarzet/SpotiFLAC-Android/releases)

- **Android**: Download `SpotiFLAC-vX.X.X-android.apk`
- **iOS**: Download `SpotiFLAC-vX.X.X-ios-unsigned.ipa` (requires sideloading)

### Requirements

**Android**
- Android 7.0 (API 24) or higher
- Storage permission for saving music files

**iOS**
- iOS 14.0 or higher
- Sideloading tool (AltStore, Sideloadly, etc.)

## Building from Source

### Prerequisites
- Flutter 3.24.0 or higher
- Go 1.21 or higher
- gomobile (`go install golang.org/x/mobile/cmd/gomobile@latest`)

### Android Build

```bash
# Build Go backend
cd go_backend
gomobile bind -target=android -androidapi 24 -o ../android/app/libs/gobackend.aar .
cd ..

# Build APK
flutter build apk --release
```

### iOS Build

#### Option 1: Using GitHub Actions (Recommended - No Mac Required)
Push to the repository and GitHub Actions will automatically build the iOS app.
Download the unsigned IPA from the Actions artifacts.

#### Option 2: Local Build (Requires macOS)

```bash
# Build Go backend for iOS
cd go_backend
gomobile bind -target=ios -o ../ios/Frameworks/Gobackend.xcframework .
cd ..

# Build iOS (unsigned)
flutter build ios --release --no-codesign
```

## Project Structure

```
SpotiFLAC-Android/
├── lib/                    # Flutter/Dart code
│   ├── models/             # Data models
│   ├── providers/          # Riverpod state management
│   ├── screens/            # UI screens
│   ├── services/           # Platform bridge & FFmpeg
│   └── theme/              # Material 3 theming
├── go_backend/             # Go backend (Tidal, Qobuz, Amazon APIs)
├── android/                # Android platform code
├── ios/                    # iOS platform code
└── .github/workflows/      # CI/CD workflows
```

## Creating a Release

Releases are automated via GitHub Actions. To create a new release:

1. Create and push a tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically:
   - Build Android APK
   - Build iOS IPA (unsigned)
   - Create a GitHub Release with both artifacts

## Known Limitations

- iOS IPA is unsigned and requires sideloading
- TestFlight distribution requires Apple Developer account ($99/year)
- Some streaming services may have regional restrictions

## License

Private project - not for public distribution.

## Disclaimer

This project is for educational purposes only. Please respect copyright laws and the terms of service of streaming platforms.
