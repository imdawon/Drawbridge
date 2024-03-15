GOARCH=amd64 GOOS=windows go build -o release/Drawbridge_windows_amd64_$1.exe -ldflags="-s -w" .
cp -R ./cmd/dashboard/ui/static ./release/
tar -zcvf ./release/Drawbridge_windows_amd64_$1.zip release/static/ release/Drawbridge_windows_amd64_$1.exe
gpg --output release/Drawbridge_windows_amd64_$1.zip.sig --detach-sig release/Drawbridge_windows_amd64_$1.exe
