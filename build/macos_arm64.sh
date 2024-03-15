GOARCH=arm64 GOOS=darwin go build -o release/Drawbridge_macos_arm64_$1 -ldflags="-s -w" .
cp -R ./cmd/dashboard/ui/static ./release/
tar -zcvf ./release/Drawbridge_macos_arm64_$1.zip release/static/ release/Drawbridge_macos_arm64_$1
gpg --output release/Drawbridge_macos_arm64_$1.zip.sig --detach-sig release/Drawbridge_macos_arm64_$1.zip
