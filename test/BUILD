load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

py_binary(
    name = "test",
    srcs = ["test.py"],
)

go_library(
    name = "runner_lib",
    srcs = ["runner.go"],
    importpath = "eval/runner",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "runner",
    embed = [":runner_lib"],
    visibility = ["//visibility:public"],
)

# container_push(
#    name = "push_foo",
#    image = ":foo",
#    format = "Docker",
#    registry = "gcr.io",
#    repository = "my-project/my-image",
#    tag = "dev",
# )
