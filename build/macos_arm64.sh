GOARCH=arm64 GOOS=darwin go build -trimpath -o release/Drawbridge_macos_arm64_$1 -ldflags="-s -w" .
cp -R ./cmd/dashboard/ui/static ./release/
zip -r ./release/Drawbridge_macos_arm64_$1.zip release/static/ release/Drawbridge_macos_arm64_$1
gpg --armor --output release/Drawbridge_macos_arm64_$1.zip.asc --detach-sig release/Drawbridge_macos_arm64_$1.zip
