workspace(name = "com_github_tsuri_eval")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "d6b2513456fe2229811da7eb67a444be7785f5323c6708b38d851d2b51e54d83",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.30.0/rules_go-v0.30.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.30.0/rules_go-v0.30.0.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "de69a09dc70417580aabf20a28619bb3ef60d038470c7cf8442fafcf627c21cb",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

#go_register_toolchains(version = "1.17.6")
go_register_toolchains(version = "1.18.2")


load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

############################################################
# Define your own dependencies here using go_repository.
# Else, dependencies declared by rules_go/gazelle will be used.
# The first declaration of an external repository "wins".
############################################################
load("//:deps.bzl", "go_dependencies")

# gazelle:repository_macro deps.bzl%go_dependencies
go_dependencies()

gazelle_dependencies()

# This commit is tagged as v1.3
bazel_gomock_commit = "fde78c91cf1783cc1e33ba278922ba67a6ee2a84"
http_archive(
    name = "bazel_gomock",
    sha256 = "692421b0c5e04ae4bc0bfff42fb1ce8671fe68daee2b8d8ea94657bb1fcddc0a",
    strip_prefix = "bazel_gomock-{v}".format(v = bazel_gomock_commit),
    urls = [
        "https://github.com/jmhodges/bazel_gomock/archive/{v}.tar.gz".format(v = bazel_gomock_commit),
    ],
)

# rules_proto defines abstract rules for building Protocol Buffers.
# This was working, tried a newer one for imprting google well known protos.
# Didn't work for that, but the newer version seems ok, so keeping it for now.
# http_archive(
#     name = "rules_proto",
#     sha256 = "2490dca4f249b8a9a3ab07bd1ba6eca085aaf8e45a734af92aad0c42d9dc7aaf",
#     strip_prefix = "rules_proto-218ffa7dfa5408492dc86c01ee637614f8695c45",
#     urls = [
#         "https://github.com/bazelbuild/rules_proto/archive/218ffa7dfa5408492dc86c01ee637614f8695c45.tar.gz",
#     ],
# )
http_archive(
    name = "rules_proto",
    sha256 = "66bfdf8782796239d3875d37e7de19b1d94301e8972b3cbd2446b332429b4df1",
    strip_prefix = "rules_proto-4.0.0",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_proto/archive/refs/tags/4.0.0.tar.gz",
        "https://github.com/bazelbuild/rules_proto/archive/refs/tags/4.0.0.tar.gz",
    ],
)
load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")
rules_proto_dependencies()
rules_proto_toolchains()

#### rules docker

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "85ffff62a4c22a74dbd98d05da6cf40f497344b3dbf1e1ab0a37ab2a1a6ca014",
    strip_prefix = "rules_docker-0.23.0",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.23.0/rules_docker-v0.23.0.tar.gz"],
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)
container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)

container_pull(
    name = "distroless_base_debian10",
    registry = "gcr.io",
    repository = "distroless/base-debian10",
    # 'tag' is also supported, but digest is encouraged for reproducibility.
    # Find the SHA256 digest value from the detials page of prebuilt containers.
    # https://console.cloud.google.com/gcr/images/distroless/GLOBAL/base-debian10
    digest = "sha256:732acc54362badaa64d9c01619020cf96ce240b97e2f1390d2a44cc22b9ba6a3",
)

# for debug
container_pull(
    name = "distroless_base_debian10_debug",
    registry = "gcr.io",
    repository = "distroless/base-debian10",
    tag = "debug",
    # 'tag' is also supported, but digest is encouraged for reproducibility.
    # Find the SHA256 digest value from the detials page of prebuilt containers.
    # https://console.cloud.google.com/gcr/images/distroless/GLOBAL/base-debian10
    digest = "sha256:8ca4526452afe5d03f53c41c76c4ddb079734eb99913aff7069bfd0d72457726",
)


# This requires rules_docker to be fully instantiated before
# it is pulled in.
# Download the rules_k8s repository
RULES_K8S_VER="0.4"
RULES_K8S_HASH="d91aeb17bbc619e649f8d32b65d9a8327e5404f451be196990e13f5b7e2d17bb"

http_archive(
    name = "io_bazel_rules_k8s",
    sha256 = RULES_K8S_HASH,
    strip_prefix = "rules_k8s-%s" % RULES_K8S_VER,
    urls = ["https://github.com/bazelbuild/rules_k8s/releases/download/v%s/rules_k8s-v%s.tar.gz" % (RULES_K8S_VER, RULES_K8S_VER)],
)

load("@io_bazel_rules_k8s//k8s:k8s.bzl", "k8s_repositories", "k8s_defaults")

k8s_repositories()

load("@io_bazel_rules_k8s//k8s:k8s_go_deps.bzl", k8s_go_deps = "deps")

k8s_go_deps()

load(
    "@io_bazel_rules_docker//python:image.bzl",
    _py_image_repos = "repositories",
)

_py_image_repos()

#### rules nodejs
http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "ee3280a7f58aa5c1caa45cb9e08cbb8f4d74300848c508374daf37314d5390d6",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/5.5.1/rules_nodejs-5.5.1.tar.gz"],
)

load("@build_bazel_rules_nodejs//:repositories.bzl", "build_bazel_rules_nodejs_dependencies")

build_bazel_rules_nodejs_dependencies()

load("@build_bazel_rules_nodejs//:index.bzl", "yarn_install")

yarn_install(
    # Name this npm so that Bazel Label references look like @npm//package
    name = "npm",
    package_json = "//:package.json",
    yarn_lock = "//:yarn.lock",
)


