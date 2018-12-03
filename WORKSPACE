workspace(name = "bazel_buildbarn")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_gomock",
    sha256 = "2f8df5baab5ed7d7aaa4c73a7789e468b675c344312cb03c3d9b677563cac535",
    strip_prefix = "bazel_gomock-5b73edb74e569ff404b3beffc809d6d9f205e0e4",
    urls = ["https://github.com/jmhodges/bazel_gomock/archive/5b73edb74e569ff404b3beffc809d6d9f205e0e4.tar.gz"],
)

http_archive(
    name = "bazel_toolchains",
    sha256 = "b5855c9a7635a877ba0c5c939395258f6a6b202901dd9f545018d57aba31b72c",
    strip_prefix = "bazel-toolchains-6e25ae44ec4a3b14fea9506633f3f820f3b76224",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/archive/6e25ae44ec4a3b14fea9506633f3f820f3b76224.tar.gz",
        "https://github.com/bazelbuild/bazel-toolchains/archive/6e25ae44ec4a3b14fea9506633f3f820f3b76224.tar.gz",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "29d109605e0d6f9c892584f07275b8c9260803bf0c6fcb7de2623b2bedc910bd",
    strip_prefix = "rules_docker-0.5.1",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.5.1.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "f87fa87475ea107b3c69196f39c82b7bbf58fe27c62a338684c20ca17d1d8613",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.16.2/rules_go-0.16.2.tar.gz"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "bc653d3e058964a5a26dcad02b6c72d7d63e6bb88d94704990b908a1445b8758",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.13.0/bazel-gazelle-0.13.0.tar.gz"],
)

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
    container_repositories = "repositories",
)

container_repositories()

container_pull(
    name = "rbe_debian8_base",
    digest = "sha256:75ba06b78aa99e58cfb705378c4e3d6f0116052779d00628ecb73cd35b5ea77d",
    registry = "launcher.gcr.io",
    repository = "google/rbe-debian8",
)

load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "com_github_aws_aws_sdk_go",
    importpath = "github.com/aws/aws-sdk-go",
    sha256 = "b6cd9c78df8aeb973f8d9b01d11c1d1e5096850614b3a3e0b4111ec747d811d3",
    strip_prefix = "aws-sdk-go-bc3f534c19ffdf835e524e11f0f825b3eaf541c3",
    urls = ["https://github.com/aws/aws-sdk-go/archive/bc3f534c19ffdf835e524e11f0f825b3eaf541c3.tar.gz"],
)

go_repository(
    name = "com_github_beorn7_perks",
    commit = "3a771d992973f24aa725d07868b467d1ddfceafb",
    importpath = "github.com/beorn7/perks",
)

go_repository(
    name = "com_github_go_ini_ini",
    commit = "358ee7663966325963d4e8b2e1fbd570c5195153",
    importpath = "github.com/go-ini/ini",
)

go_repository(
    name = "com_github_golang_protobuf",
    commit = "b4deda0973fb4c70b50d226b1af49f3da59f5265",
    importpath = "github.com/golang/protobuf",
)

go_repository(
    name = "com_github_jmespath_go_jmespath",
    commit = "0b12d6b5",
    importpath = "github.com/jmespath/go-jmespath",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    commit = "c12348ce28de40eed0136aa2b644d0ee0650e56c",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
)

go_repository(
    name = "com_github_prometheus_client_golang",
    commit = "c5b7fccd204277076155f10851dad72b76a49317",
    importpath = "github.com/prometheus/client_golang",
)

go_repository(
    name = "com_github_prometheus_client_model",
    commit = "5c3871d89910bfb32f5fcab2aa4b9ec68e65a99f",
    importpath = "github.com/prometheus/client_model",
)

go_repository(
    name = "com_github_prometheus_common",
    commit = "7600349dcfe1abd18d72d3a1770870d9800a7801",
    importpath = "github.com/prometheus/common",
)

go_repository(
    name = "com_github_prometheus_procfs",
    commit = "ae68e2d4c00fed4943b5f6698d504a5fe083da8a",
    importpath = "github.com/prometheus/procfs",
)

go_repository(
    name = "com_github_satori_go_uuid",
    commit = "f58768cc1a7a7e77a3bd49e98cdd21419399b6a3",
    importpath = "github.com/satori/go.uuid",
)

go_repository(
    name = "org_golang_google_genproto",
    commit = "e92b116572682a5b432ddd840aeaba2a559eeff1",
    importpath = "google.golang.org/genproto",
)

go_repository(
    name = "org_golang_google_grpc",
    commit = "168a6198bcb0ef175f7dacec0b8691fc141dc9b8",
    importpath = "google.golang.org/grpc",
)

go_repository(
    name = "org_golang_x_net",
    commit = "039a4258aec0ad3c79b905677cceeab13b296a77",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "org_golang_x_text",
    commit = "f21a4dfb5e38f5895301dc265a8def02365cc3d0",
    importpath = "golang.org/x/text",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_prometheus",
    commit = "c225b8c3b01faf2899099b768856a9e916e5087b",
    importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
)

go_repository(
    name = "com_github_go_redis_redis",
    commit = "480db94d33e6088e08d628833b6c0705451d24bb",
    importpath = "github.com/go-redis/redis",
)

go_repository(
    name = "com_github_bazelbuild_remote_apis",
    importpath = "github.com/bazelbuild/remote-apis",
    sha256 = "99ab1378f10854504c75bcfa43be2129d36bbba8e80a79a4216a3e3026a0985b",
    strip_prefix = "remote-apis-ed4849810292e5fb3c844992133523f01a4ad420",
    urls = ["https://github.com/bazelbuild/remote-apis/archive/ed4849810292e5fb3c844992133523f01a4ad420.tar.gz"],
)

go_repository(
    name = "com_github_golang_mock",
    importpath = "github.com/golang/mock",
    sha256 = "0dc7dbcf6d83b4318e26d9481dfa9405042288d666835f810e0b70ada2f54e11",
    strip_prefix = "mock-aedf487a10d1285646a046e4c9537d7854e820e1",
    urls = ["https://github.com/EdSchouten/mock/archive/aedf487a10d1285646a046e4c9537d7854e820e1.tar.gz"],
)

go_repository(
    name = "com_github_stretchr_testify",
    commit = "04af85275a5c7ac09d16bb3b9b2e751ed45154e5",
    importpath = "github.com/stretchr/testify",
)

go_repository(
    name = "com_github_gorilla_context",
    commit = "08b5f424b9271eedf6f9f0ce86cb9396ed337a42",
    importpath = "github.com/gorilla/context",
)

go_repository(
    name = "com_github_gorilla_mux",
    commit = "e3702bed27f0d39777b0b37b664b6280e8ef8fbf",
    importpath = "github.com/gorilla/mux",
)

go_repository(
    name = "com_github_kballard_go_shellquote",
    commit = "95032a82bc518f77982ea72343cc1ade730072f0",
    importpath = "github.com/kballard/go-shellquote",
)

go_repository(
    name = "com_github_buildkite_terminal",
    importpath = "github.com/buildkite/terminal",
    sha256 = "ba8e7566a7fe5c62089fa56191c80bf15b121ae369d0c17f6f80669d3bbbb23f",
    strip_prefix = "terminal-3.1.0",
    urls = ["https://github.com/buildkite/terminal/archive/v3.1.0.tar.gz"],
)

go_repository(
    name = "com_github_google_uuid",
    importpath = "github.com/google/uuid",
    sha256 = "7e330758f7c81d9f489493fb7ae0e67d06f50753429758b64f25ad5fb2727e21",
    strip_prefix = "uuid-1.1.0",
    urls = ["https://github.com/google/uuid/archive/v1.1.0.tar.gz"],
)
