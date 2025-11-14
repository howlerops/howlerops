# typed: false
# frozen_string_literal: true

# HowlerOps Homebrew Formula
# Usage: brew install sql-studio/tap/howlerops

class Howlerops < Formula
  desc "Native HowlerOps desktop SQL client"
  homepage "https://github.com/sql-studio/sql-studio"
  version "2.0.0"
  license "MIT"

  on_intel do
    url "https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/howlerops-darwin-universal.tar.gz"
    sha256 "PLACEHOLDER_DARWIN_SHA256"
  end

  on_arm do
    url "https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/howlerops-darwin-universal.tar.gz"
    sha256 "PLACEHOLDER_DARWIN_SHA256"
  end

  def install
    prefix.install "howlerops.app"
    bin.install_symlink "#{prefix}/howlerops.app/Contents/MacOS/howlerops" => "howlerops"
  end

  def caveats
    <<~EOS
      HowlerOps.app was installed to:
        #{prefix}/howlerops.app

      Launch the application with:
        open -a HowlerOps
      Or start it from the terminal:
        howlerops
    EOS
  end

  test do
    system "#{bin}/howlerops", "--help"
  end
end
