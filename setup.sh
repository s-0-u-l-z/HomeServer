#!/bin/bash

set -e

echo "🔍 Checking for Go..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed."

    if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" || "$OSTYPE" == "cygwin" ]]; then
        echo "➡️ You're on Windows. Install Go from:"
        echo "https://go.dev/dl/"
        exit 1
    fi

    echo "🛠 What package manager are you using? (apt, dnf, pacman, etc.):"
    read -r PKG

    echo "📦 Installing Go using $PKG..."
    case "$PKG" in
        apt)
            sudo apt update && sudo apt install -y golang
            ;;
        dnf)
            sudo dnf install -y golang
            ;;
        pacman)
            sudo pacman -Syu --noconfirm go
            ;;
        zypper)
            sudo zypper install -y go
            ;;
        *)
            echo "❌ Unsupported package manager: $PKG"
            exit 1
            ;;
    esac
else
    echo "✅ Go is already installed."
fi

echo "🔍 Checking for OpenSSL..."
if ! command -v openssl &> /dev/null; then
    echo "❌ OpenSSL is not installed."

    if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" || "$OSTYPE" == "cygwin" ]]; then
        echo "➡️ On Windows, install OpenSSL using Chocolatey or manually:"
        echo "https://slproweb.com/products/Win32OpenSSL.html"
        exit 1
    fi

    echo "📦 Installing OpenSSL using $PKG..."
    case "$PKG" in
        apt)
            sudo apt install -y openssl
            ;;
        dnf)
            sudo dnf install -y openssl
            ;;
        pacman)
            sudo pacman -Syu --noconfirm openssl
            ;;
        zypper)
            sudo zypper install -y libopenssl1_1
            ;;
        *)
            echo "❌ Unsupported package manager for OpenSSL: $PKG"
            exit 1
            ;;
    esac
else
    echo "✅ OpenSSL is already installed."
fi

echo "🎉 All requirements are installed. You can now run your HomeServer."
