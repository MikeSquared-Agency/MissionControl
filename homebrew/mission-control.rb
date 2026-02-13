class MissionControl < Formula
  desc "AI agent orchestration CLI for Claude Code"
  homepage "https://github.com/MikeSquared-Agency/MissionControl"
  version "0.5.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/MikeSquared-Agency/MissionControl/releases/download/v#{version}/mission-control-#{version}-darwin-arm64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
    on_intel do
      url "https://github.com/MikeSquared-Agency/MissionControl/releases/download/v#{version}/mission-control-#{version}-darwin-amd64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/MikeSquared-Agency/MissionControl/releases/download/v#{version}/mission-control-#{version}-linux-arm64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
    on_intel do
      url "https://github.com/MikeSquared-Agency/MissionControl/releases/download/v#{version}/mission-control-#{version}-linux-amd64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  depends_on "tmux"

  def install
    bin.install "mc"
    bin.install "mc-core"
    bin.install "mc-orchestrator"
  end

  def caveats
    <<~EOS
      MissionControl has been installed!

      To get started:
        1. Navigate to your project directory
        2. Run: mc init
        3. Run: mc serve

      For more information, visit:
        https://github.com/MikeSquared-Agency/MissionControl
    EOS
  end

  test do
    system "#{bin}/mc", "--help"
    system "#{bin}/mc-core", "--help"
  end
end
