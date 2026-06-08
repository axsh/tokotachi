class Tt < Formula
  desc "Tokotachi CLI - Project scaffolding and management tool"
  homepage "https://github.com/axsh/tokotachi"
  version "0.6.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.1/tt_darwin_arm64.tar.gz"
      sha256 "1d098eeb2a54cb488304acdc6c7d36457948dc5666ce09b17f17125776db724d"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.1/tt_darwin_amd64.tar.gz"
      sha256 "ea646e3877b05b8f99ab2042b9d2cbfea5a7e071c2a613a42f7434304a6f1163"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.1/tt_linux_arm64.tar.gz"
      sha256 "d9feda75088413e11e9e05b38fbf6510faab2ceae4364f9b2db94befab534844"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.1/tt_linux_amd64.tar.gz"
      sha256 "81e9fbef56553611d2f416128a2bcba0e3c3c6d841ae3dc09ab492347604e7fa"
    end
  end

  def install
    bin.install "tt"
  end

  test do
    system "#{bin}/tt", "version"
  end
end
