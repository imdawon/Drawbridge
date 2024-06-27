GOARCH=amd64 GOOS=linux go build -o release/Drawbridge_linux_amd64_$1 -ldflags="-s -w -trimpath" .
cp -R ./cmd/dashboard/ui/static ./release/
zip -r ./release/Drawbridge_linux_amd64_$1.zip release/static/ release/Drawbridge_linux_amd64_$1
gpg --armor --output release/Drawbridge_linux_amd64_$1.zip.asc --detach-sig release/Drawbridge_linux_amd64_$1.zip
