# © 2026 Anna Veretennykova · ANNÁVE TECH · https://www.annave.tech
class Annave < Formula
  desc "ANNÁVE developer tools — log analysis, health checks, docs, and cleanup"
  homepage "https://www.annave.tech"
  version "0.1.0"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/annavetech/annave-cli-golang/releases/download/v#{version}/annave-darwin-arm64"
      sha256 "PLACEHOLDER_DARWIN_ARM64"
    else
      url "https://github.com/annavetech/annave-cli-golang/releases/download/v#{version}/annave-darwin-amd64"
      sha256 "PLACEHOLDER_DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/annavetech/annave-cli-golang/releases/download/v#{version}/annave-linux-arm64"
      sha256 "PLACEHOLDER_LINUX_ARM64"
    else
      url "https://github.com/annavetech/annave-cli-golang/releases/download/v#{version}/annave-linux-amd64"
      sha256 "PLACEHOLDER_LINUX_AMD64"
    end
  end

  def install
    bin.install Dir["annave-*"].first => "annave"
  end

  test do
    assert_match "annave version #{version}", shell_output("#{bin}/annave version")
  end
end
