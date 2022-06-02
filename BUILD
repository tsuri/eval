load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_go//go:def.bzl", "go_embed_data", "go_library")

##load("@io_bazel_rules_docker//container:container.bzl", "container_bundle")
#load("@io_bazel_rules_docker//contrib:push-all.bzl", "container_push")
load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

# gazelle:prefix eval
gazelle(name = "gazelle")

gazelle(
    name = "gazelle-update-repos",
    args = [
        "-from_file=go.mod",
        "-to_macro=deps.bzl%go_dependencies",
        "-prune",
        "-build_file_proto_mode=disable_global",
    ],
    command = "update-repos",
)

filegroup(
    name = "evaluations",
    srcs = glob(["evaluations/**/*.star"]),
    visibility = ["//visibility:public"],
)

k8s_objects(
    name = "eval",
    objects = [
        "//cmd/cache:dev",
        "//cmd/engine:dev",
        "//cmd/runner:dev",
        "//cmd/builder:dev",
    ],
)

# genrule(
#     name = "stamper",
#     outs = ["stamper.out"],
#     srcs = [
#         ".git/HEAD",
#         ".git/refs/heads/main",
#     ],
#     cmd = """
# if [[ $$(cat $(location :.git/HEAD)) = "refs: refs/heads/<release branch>" ]]; then
#   cat $(location :.git/refs/heads/main) > stamper.out
# else
#   # If we're not on the release branch, don't uncache things on commit.
#   echo "dev" > stamper.out
# fi
# """,
# )

genrule(
    name = "build_info_data",
    srcs = ["bazel-out/stable-status.txt"],
    outs = ["build_info.go"],
    cmd = "./$(location :scripts/gen_version) $(location :bazel-out/stable-status.txt)> \"$@\"",
    tools = ["scripts/gen_version"],
)

go_embed_data(
    name = "binfo_embed",
    src = ":build_info_data",
    package = "binfo",
    var = "greeting",
)

go_library(
    name = "binfo",
    srcs = [":binfo_embed"],
    importpath = "eval/binfo",
    visibility = ["//visibility:public"],
)
