# Homebrew Formula for ccAgents
# This formula will be automatically updated by the release workflow

class Ccagents < Formula
  desc "AI-powered GitHub issue-to-PR automation tool"
  homepage "https://github.com/fumiya-kume/cca"
  version "1.0.0"  # This will be updated automatically
  license "MIT"

  if Hardware::CPU.intel?
    url "https://github.com/fumiya-kume/cca/releases/download/v#{version}/ccagents-darwin-amd64.tar.gz"
    sha256 "sha256-placeholder-amd64"  # This will be updated automatically
  elsif Hardware::CPU.arm?
    url "https://github.com/fumiya-kume/cca/releases/download/v#{version}/ccagents-darwin-arm64.tar.gz"
    sha256 "sha256-placeholder-arm64"  # This will be updated automatically
  end

  depends_on "gh"      # GitHub CLI (required)
  
  def install
    bin.install "ccagents-darwin-#{Hardware::CPU.arch}" => "ccagents"
    
    # Install shell completions
    generate_completions_from_executable(bin/"ccagents", "completion")
    
    # Install man page (if available)
    # man1.install "docs/ccagents.1" if File.exist?("docs/ccagents.1")
  end

  def caveats
    <<~EOS
      ccAgents requires authentication with both GitHub and Claude:
      
      1. GitHub authentication:
         gh auth login
      
      2. Claude authentication:
         # Visit https://claude.ai/code for setup instructions
      
      3. Initialize ccAgents in your project:
         ccagents init
      
      4. Validate your setup:
         ccagents validate
      
      For more information:
         ccagents help
         https://github.com/fumiya-kume/cca/blob/main/docs/getting-started.md
    EOS
  end

  test do
    # Test that the binary exists and can show version
    assert_match version.to_s, shell_output("#{bin}/ccagents version --short")
    
    # Test help command
    assert_match "ccAgents", shell_output("#{bin}/ccagents help")
    
    # Test config validation (should fail without config)
    assert_match "validation", shell_output("#{bin}/ccagents validate", 1)
  end
end