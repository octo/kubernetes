package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "fake_authorization_client.go",
        "fake_generated_expansion.go",
        "fake_localsubjectaccessreview.go",
        "fake_localsubjectaccessreview_expansion.go",
        "fake_selfsubjectaccessreview.go",
        "fake_selfsubjectaccessreview_expansion.go",
        "fake_subjectaccessreview.go",
        "fake_subjectaccessreview_expansion.go",
    ],
    tags = ["automanaged"],
    deps = [
        "//pkg/apis/authorization/v1beta1:go_default_library",
        "//pkg/client/clientset_generated/clientset/typed/authorization/v1beta1:go_default_library",
        "//pkg/client/restclient:go_default_library",
        "//pkg/client/testing/core:go_default_library",
    ],
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [":package-srcs"],
    tags = ["automanaged"],
)
