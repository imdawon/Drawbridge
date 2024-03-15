GOARCH=amd64 GOOS=linux go build -o release/Drawbridge_linux_amd64_$1 -ldflags="-s -w" .
cp -R ./cmd/dashboard/ui/static ./release/
tar -zcvf ./release/Drawbridge_linux_amd64_$1.zip release/static/ release/Drawbridge_linux_amd64_$1
gpg --output release/Drawbridge_linux_amd64_$1.zip.sig --detach-sig release/Drawbridge_linux_amd64_$1.zip
