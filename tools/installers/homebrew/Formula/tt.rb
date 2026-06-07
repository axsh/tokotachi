class Tt < Formula
  desc "Tokotachi CLI - Project scaffolding and management tool"
  homepage "https://github.com/axsh/tokotachi"
  version "0.6.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.0/tt_darwin_arm64.tar.gz"
      sha256 "c1788b9df55ac31464f419d7fd0dfd072a5d66f87743defd8c87c88eab2fcd0c"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.0/tt_darwin_amd64.tar.gz"
      sha256 "eba04dddf1124bcd18ec83bdb59fd26f00e059a4bafc47aa493f8df2fb977e9f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.0/tt_linux_arm64.tar.gz"
      sha256 "de37f3f7793bfaba6444aa112f23a0e5c3c688555252ae5c9f28dd42f3073500"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.6.0/tt_linux_amd64.tar.gz"
      sha256 "e792338173aeff68131ae01be707c6bc1963234f2b67fc3378f50859c8966574"
    end
  end

  def install
    bin.install "tt"
  end

  test do
    system "#{bin}/tt", "version"
  end
end
