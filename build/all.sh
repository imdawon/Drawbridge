#!/bin/zsh

rm -rf release/
echo -n "Enter Version (xxx) with -alpha or -beta tag if relevant: "
read version
./build/linux_amd64.sh $(echo $version)
./build/macos_arm64.sh $(echo $version)
./build/windows_amd64.sh $(echo $version)
