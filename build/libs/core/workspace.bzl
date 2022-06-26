#load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def require(repo_rule, name, **kwargs):
    """Defines a repository if it does not already exist.
    """
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
