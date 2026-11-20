class Tt < Formula
  desc "Tokotachi CLI - Project scaffolding and management tool"
  homepage "https://github.com/axsh/tokotachi"
  version "0.9.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.2/tt_darwin_arm64.tar.gz"
      sha256 "df6fc7e4c0020d279cf565545da4aa0c9d205b78bc8546327061803bcd0e4923"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.2/tt_darwin_amd64.tar.gz"
      sha256 "58df68be93b6392f1890683760b12cd33c8d69971073b0750a24fd81248c8df2"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.2/tt_linux_arm64.tar.gz"
      sha256 "7f68d736a6d44e087ce0fa27d1c2ce006a3e992dab0577f39ae9aa69c8b5f6e3"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.2/tt_linux_amd64.tar.gz"
      sha256 "b766ae18c449330824c896cfe9ca8efc252cdcfe340c7b9975feb4d75295f7eb"
    end
  end

  def install
    bin.install "tt"
  end

  test do
    system "#{bin}/tt", "version"
  end
end
