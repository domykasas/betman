#!/bin/bash

# Install dependencies for P2P Coin Flip Betting Game

set -e

echo "Installing dependencies for P2P Coin Flip Betting Game..."

# Detect OS
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    echo "Detected Linux OS"
    
    if command -v apt-get &> /dev/null; then
        # Debian/Ubuntu
        echo "Installing dependencies with apt-get..."
        sudo apt-get update
        sudo apt-get install -y \
            build-essential \
            pkg-config \
            libgl1-mesa-dev \
            xorg-dev \
            libx11-dev \
            libxrandr-dev \
            libxinerama-dev \
            libxcursor-dev \
            libxi-dev \
            libxxf86vm-dev
    elif command -v dnf &> /dev/null; then
        # Fedora/RHEL 8+
        echo "Installing dependencies with dnf..."
        sudo dnf install -y \
            gcc \
            gcc-c++ \
            make \
            pkgconfig \
            mesa-libGL-devel \
            libX11-devel \
            libXrandr-devel \
            libXinerama-devel \
            libXcursor-devel \
            libXi-devel
    elif command -v yum &> /dev/null; then
        # CentOS/RHEL 7
        echo "Installing dependencies with yum..."
        sudo yum install -y \
            gcc \
            gcc-c++ \
            make \
            pkgconfig \
            mesa-libGL-devel \
            libX11-devel \
            libXrandr-devel \
            libXinerama-devel \
            libXcursor-devel \
            libXi-devel
    elif command -v pacman &> /dev/null; then
        # Arch Linux
        echo "Installing dependencies with pacman..."
        sudo pacman -S --needed \
            base-devel \
            pkg-config \
            mesa \
            libx11 \
            libxrandr \
            libxinerama \
            libxcursor \
            libxi
    else
        echo "Unsupported Linux distribution. Please install development tools and X11 libraries manually."
        exit 1
    fi
    
elif [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    echo "Detected macOS"
    
    if ! command -v brew &> /dev/null; then
        echo "Homebrew not found. Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi
    
    echo "Installing dependencies with Homebrew..."
    brew install pkg-config
    
    # Check if Xcode command line tools are installed
    if ! xcode-select -p &> /dev/null; then
        echo "Installing Xcode command line tools..."
        xcode-select --install
        echo "Please complete the Xcode command line tools installation and run this script again."
        exit 0
    fi
    
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    # Windows
    echo "Detected Windows"
    
    if command -v choco &> /dev/null; then
        echo "Installing dependencies with Chocolatey..."
        choco install mingw make -y
    else
        echo "Chocolatey not found. Please install mingw-w64 and make manually."
        echo "You can download mingw-w64 from: https://www.mingw-w64.org/"
        exit 1
    fi
    
else
    echo "Unsupported operating system: $OSTYPE"
    exit 1
fi

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go 1.19 or later from https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.19"

if [[ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]]; then
    echo "Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or later."
    exit 1
fi

echo "Go version $GO_VERSION is sufficient."

# Install Fyne dependencies
echo "Installing Fyne packaging tool..."
go install fyne.io/fyne/v2/cmd/fyne@latest

echo "Dependencies installed successfully!"
echo ""
echo "You can now build the application with:"
echo "  make build"
echo ""
echo "Or run it directly with:"
echo "  make run"