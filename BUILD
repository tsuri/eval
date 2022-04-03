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

#container_bundle(
#    name = "all_container",
#    images = {
#        "gcr.io/$(project_id)/$(repo)/client": "//client:container",
#        "gcr.io/$(project_id)/$(repo)/server": "//server:container",
#    },
#)

##container_push(
#    name = "push_all",
#    bundle = ":all_container",
#    format = "Docker",
#)

#k8s_objects(
#   name = "gke_deploy",
#   objects = [
#      "//server:gke_deploy",
#      "//client:gke_deploy",
#   ],
#)

# k8s_object(
#     name = "namespace",
#     # cluster as in `kubectl config view --minify -o=jsonpath='{.contexts[0].context.cluster}'`
#     cluster = "kind-eval",
#     context = "kind-eval",
#     kind = "namespace",
#     # A template of a Kubernetes Deployment object yaml.
#     template = "//deployments/namespaces.yaml",
# )

k8s_objects(
    name = "eval",
    objects = [
        "//cmd/engine:dev",
        "//cmd/grunt:dev",
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
