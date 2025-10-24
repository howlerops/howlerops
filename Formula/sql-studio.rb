# typed: false
# frozen_string_literal: true

# SQL Studio Homebrew Formula
# This formula allows users to install SQL Studio via Homebrew
# Usage: brew install sql-studio/tap/sql-studio

class SqlStudio < Formula
  desc "Modern SQL database client with cloud sync capabilities"
  homepage "https://github.com/sql-studio/sql-studio"
  version "2.0.0"
  license "MIT"

  # Intel (x86_64) binary
  on_intel do
    url "https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/sql-studio-darwin-amd64.tar.gz"
    sha256 "PLACEHOLDER_AMD64_SHA256"
  end

  # Apple Silicon (ARM64) binary
  on_arm do
    url "https://github.com/sql-studio/sql-studio/releases/download/v2.0.0/sql-studio-darwin-arm64.tar.gz"
    sha256 "PLACEHOLDER_ARM64_SHA256"
  end

  # Installation steps
  def install
    # Install the binary to Homebrew's bin directory
    bin.install "sql-studio"

    # Generate and install shell completions if available
    if File.exist?("completions")
      bash_completion.install "completions/sql-studio.bash" if File.exist?("completions/sql-studio.bash")
      fish_completion.install "completions/sql-studio.fish" if File.exist?("completions/sql-studio.fish")
      zsh_completion.install "completions/_sql-studio" if File.exist?("completions/_sql-studio")
    end

    # Install man pages if available
    man1.install Dir["man/*.1"] if Dir.exist?("man")
  end

  # Post-installation message
  def caveats
    <<~EOS
      SQL Studio has been installed successfully!

      To get started, run:
        sql-studio

      For help and documentation, visit:
        https://github.com/sql-studio/sql-studio

      To check the version:
        sql-studio --version

      Note: SQL Studio stores its configuration in:
        ~/.config/sql-studio/
    EOS
  end

  # Test block to verify installation
  test do
    # Verify the binary exists and is executable
    assert_predicate bin/"sql-studio", :exist?
    assert_predicate bin/"sql-studio", :executable?

    # Test version command
    version_output = shell_output("#{bin}/sql-studio --version")
    assert_match version.to_s, version_output

    # Verify the binary runs without errors for help command
    help_output = shell_output("#{bin}/sql-studio --help")
    assert_match "SQL Studio", help_output
  end
end
