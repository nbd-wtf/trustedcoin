#!/bin/bash
set -x

parent_dir="$(dirname "$(realpath "$0")")/.."

version=$(grep -oP 'const version = "\K[^"]+' "$parent_dir/main.go")

get_platform_file_end() {
    machine=$(uname -m)
    kernel=$(uname -s)

    case $kernel in
        Linux)
            case $machine in
                x86_64)
                    echo 'linux-amd64.tar.gz'
                    ;;
                aarch64)
                    echo 'linux-arm64.tar.gz'
                    ;;
                *)
                    echo "No self-compiled binary found and unsupported release-architecture: $machine" >&2
                    exit 1
                    ;;
            esac
            ;;
        *)
            echo "No self-compiled binary found and unsupported OS: $kernel" >&2
            exit 1
            ;;
    esac
}
platform_file_end=$(get_platform_file_end)
archive_file=trustedcoin-v$version-$platform_file_end

github_url="https://github.com/nbd-wtf/trustedcoin/releases/download/v$version/$archive_file"


# Download the archive using curl
if ! curl -L "$github_url" -o "$parent_dir/$archive_file"; then
    echo "Error downloading the file from $github_url" >&2
    exit 1
fi

# Extract the contents
if ! tar -xzvf "$parent_dir/$archive_file" -C "$parent_dir"; then
    echo "Error extracting the contents of $archive_file" >&2
    exit 1
fi

