class Tt < Formula
  desc "Tokotachi CLI - Project scaffolding and management tool"
  homepage "https://github.com/axsh/tokotachi"
  version "0.9.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.3/tt_darwin_arm64.tar.gz"
      sha256 "ad8b1ff61ba903ec03d15a7aa50d55f75059931b2e2475924b0a83ceecd02d55"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.3/tt_darwin_amd64.tar.gz"
      sha256 "071b9a3fa4019d569ba8d46e8fcfd16c1557870ae30b55db79bd9333ba42b909"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.3/tt_linux_arm64.tar.gz"
      sha256 "2a07e81f2e34822c8e3bc342a564279eca3e81b2161ddad6a114a554acf1467a"
    else
      url "https://github.com/axsh/tokotachi/releases/download/v0.9.3/tt_linux_amd64.tar.gz"
      sha256 "48f2f9c5aa0f5f4720ae70cce72b516487e5211edace8381522c0dff53d4af89"
    end
  end

  def install
    bin.install "tt"
  end

  test do
    system "#{bin}/tt", "version"
  end
end
