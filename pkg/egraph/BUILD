load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["egraph.go"],
    data = [
        "//:evaluations",
    ],
    importpath = "eval/pkg/egraph",
    visibility = ["//visibility:public"],
    deps = [
        "@net_starlark_go//starlark:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["egraph_test.go"],
    embed = [":go_default_library"],
    rundir = ".",
    deps = ["@com_github_stretchr_testify//assert:go_default_library"],
)
