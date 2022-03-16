load("@bazel_gazelle//:def.bzl", "gazelle")
##load("@io_bazel_rules_docker//container:container.bzl", "container_bundle")
#load("@io_bazel_rules_docker//contrib:push-all.bzl", "container_push")
#load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

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
