class UdpSender < Formula
  desc "UDP packet sender with IP/port spoofing support"
  homepage "https://github.com/criblio/udp-sender"
  version "VERSION_PLACEHOLDER"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/criblio/udp-sender/releases/download/TAG_PLACEHOLDER/udp-sender-TAG_PLACEHOLDER-darwin-arm64.tar.gz"
      sha256 "ARM64_SHA_PLACEHOLDER"
    else
      url "https://github.com/criblio/udp-sender/releases/download/TAG_PLACEHOLDER/udp-sender-TAG_PLACEHOLDER-darwin-x64.tar.gz"
      sha256 "X64_SHA_PLACEHOLDER"
    end
  end

  def install
    if Hardware::CPU.arm?
      bin.install "udp-sender-darwin-arm64" => "udp-sender"
    else
      bin.install "udp-sender-darwin-x64" => "udp-sender"
    end
  end

  def caveats
    <<~EOS
      udp-sender requires root privileges or CAP_NET_RAW to create raw sockets.

      On macOS, run with sudo:
        sudo udp-sender [options]

      On Linux, grant capabilities instead:
        sudo setcap cap_net_raw+ep $(which udp-sender)
    EOS
  end

  test do
    assert_match "udp-sender version", shell_output("#{bin}/udp-sender --version 2>&1")
  end
end
