class Orbiton < Formula
  desc "Editor"
  homepage "https://orbiton.zip/"
  url "https://github.com/xyproto/orbiton.git",
      :tag      => "v2.64.0",
      :revision => "9798d682dc0a84d70d799fcd290e7eaa40beca4b"
  version_scheme 1
  head "https://github.com/xyproto/orbiton.git"

  depends_on "go" => :build

  def install
    ENV["GOPATH"] = buildpath
    (buildpath/"src/github.com/xyproto/orbiton").install buildpath.children
    cd "src/github.com/xyproto/orbiton/v2" do
      system "go", "build", "-o", "o"

      bin.install "o"
      prefix.install_metafiles
    end
  end

  test do
    begin
      output = shell_output("#{bin}/o", "--version")
      assert_match /Orbiton/m, output
    end
  end
end
