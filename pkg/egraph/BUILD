load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "egraph",
    srcs = ["egraph.go"],
    data = [
        "//:evaluations",
    ],
    importpath = "eval/pkg/egraph",
    visibility = ["//visibility:public"],
    deps = [
        "@net_starlark_go//starlark",
    ],
)

go_test(
    name = "egraph_test",
    srcs = ["egraph_test.go"],
    embed = [":egraph"],
    rundir = ".",
    deps = ["@com_github_stretchr_testify//assert"],
)
