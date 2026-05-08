class UdpSender < Formula
  desc "UDP packet sender with IP/port spoofing support"
  homepage "https://github.com/criblio/udp-sender"
  version "1.0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-v1.0.3-darwin-arm64.tar.gz"
      sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
    else
      url "https://github.com/criblio/udp-sender/releases/download/v1.0.3/udp-sender-v1.0.3-darwin-x64.tar.gz"
      sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
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
