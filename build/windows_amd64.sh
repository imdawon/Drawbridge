GOARCH=amd64 GOOS=windows go build -trimpath -o release/Drawbridge_windows_amd64_$1.exe -ldflags="-s -w" .
cp -R ./cmd/dashboard/ui/static ./release/
zip -r ./release/Drawbridge_windows_amd64_$1.zip release/static/ release/Drawbridge_windows_amd64_$1.exe
gpg --armor --output release/Drawbridge_windows_amd64_$1.zip.asc --detach-sig release/Drawbridge_windows_amd64_$1.zip
