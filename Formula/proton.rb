# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Proton < Formula
  desc "cli protobuf to json converter"
  homepage "https://github.com/beatlabs/proton"
  version "2.0.1"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/beatlabs/proton/releases/download/v2.0.1/proton_Darwin_x86_64.tar.gz"
      sha256 "7213d6394b8e5def43a84615c0c4c46f945aca905a8a2af63fdf9a9813edfc6f"

      def install
        bin.install "proton"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/beatlabs/proton/releases/download/v2.0.1/proton_Linux_x86_64.tar.gz"
      sha256 "5e917e5d22c75dd6362eea013023d353258a52aa5fd55351029abed245d144b9"

      def install
        bin.install "proton"
      end
    end
  end

  test do
    system "#{bin/proton}"
  end
end
