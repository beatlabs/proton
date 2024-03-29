# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Proton < Formula
  desc "cli protobuf to json converter"
  homepage "https://github.com/beatlabs/proton"
  version "2.1.0"

  on_macos do
    url "https://github.com/beatlabs/proton/releases/download/v2.1.0/proton_Darwin_x86_64.tar.gz"
    sha256 "dfcae8d7e2d67c7e0c5455a274dc7ffb3dcf5f07f3c66c518247cf3a1bdac586"

    def install
      bin.install "proton"
    end

    if Hardware::CPU.arm?
      def caveats
        <<~EOS
          The darwin_arm64 architecture is not supported for the Proton
          formula at this time. The darwin_amd64 binary may work in compatibility
          mode, but it might not be fully supported.
        EOS
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/beatlabs/proton/releases/download/v2.1.0/proton_Linux_x86_64.tar.gz"
      sha256 "35e3fdf299aacbf5baee62d755a8a4b9ff52c9f7f3380d3b7e92bf52a77b5dcf"

      def install
        bin.install "proton"
      end
    end
  end

  test do
    system "#{bin/proton}"
  end
end
