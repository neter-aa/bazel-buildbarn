workspace(name = "bazel_buildbarn")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "6dede2c65ce86289969b907f343a1382d33c14fbce5e30dd17bb59bb55bb6593",
    strip_prefix = "rules_docker-0.4.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.4.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "322cbfa381a396166ba82d7fa3513dadf8e0b069b96dedbc0c7ed0b197a81a5e",
    strip_prefix = "rules_go-7dc1d3057cdf7456cd4fbd9188e1d795e2589a70",
    urls = ["https://github.com/bazelbuild/rules_go/archive/7dc1d3057cdf7456cd4fbd9188e1d795e2589a70.tar.gz"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "bc653d3e058964a5a26dcad02b6c72d7d63e6bb88d94704990b908a1445b8758",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.13.0/bazel-gazelle-0.13.0.tar.gz"],
)

http_file(
    name = "debian_deb_bash",
    sha256 = "ec375aa432a743ec44781cb91802f5752021e7e107b48ee2e9aa2f7b9fc23f86",
    url = "http://ftp.nl.debian.org/debian/pool/main/b/bash/bash_4.4-5_amd64.deb",
)

http_file(
    name = "debian_deb_binutils",
    sha256 = "b86a5bf3ff150ef74c1a452564c6480a8f81f3f27376b121a76783dc5e59d352",
    url = "http://ftp.nl.debian.org/debian/pool/main/b/binutils/binutils_2.28-5_amd64.deb",
)

http_file(
    name = "debian_deb_coreutils",
    sha256 = "ef6c0ab3d52a7d3e85ba4a9c04a1931264d34bab842da6e1428c8c4bda28a800",
    url = "http://ftp.nl.debian.org/debian/pool/main/c/coreutils/coreutils_8.26-3_amd64.deb",
)

http_file(
    name = "debian_deb_cpp",
    sha256 = "61e8465367af69a52fe7f4300e9ea2e0b12a918a78beac41950b8a43be26aed9",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-defaults/cpp_6.3.0-4_amd64.deb",
)

http_file(
    name = "debian_deb_cpp_6",
    sha256 = "611bb72b6a432b357881a8b856fe0d9c7380b8a211cebefb489485b683949d6f",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/cpp-6_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_dash",
    sha256 = "5084b7e30fde9c51c4312f4da45d4fdfb861ab91c1d514a164dcb8afd8612f65",
    url = "http://ftp.nl.debian.org/debian/pool/main/d/dash/dash_0.5.8-2.4_amd64.deb",
)

http_file(
    name = "debian_deb_debianutils",
    sha256 = "11cdc154dc6555e093725ddcb2f6f38882f7aa7090170c07216483cc3a5964ae",
    url = "http://ftp.nl.debian.org/debian/pool/main/d/debianutils/debianutils_4.8.1.1_amd64.deb",
)

http_file(
    name = "debian_deb_findutils",
    sha256 = "b6e241e619e985455e6a7638807a28929b15efecf0158b207ff3ae0fc964f75a",
    url = "http://ftp.nl.debian.org/debian/pool/main/f/findutils/findutils_4.6.0+git+20161106-2_amd64.deb",
)

http_file(
    name = "debian_deb_gcc",
    sha256 = "64902f7486389eaf20a9ff8efaed81cb41948b43453fb6be4472418bca0a231b",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-defaults/gcc_6.3.0-4_amd64.deb",
)

http_file(
    name = "debian_deb_gcc_6",
    sha256 = "c5a6be3bc9b061ea35f33444ae063581dea2dae7eb34f960b2ae371f03b5dec7",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/gcc-6_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_gxx",
    sha256 = "3b61f34c9fa121c01287251acaed3f5754ddb83788bfc0bd899ee859e9604861",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-defaults/g++_6.3.0-4_amd64.deb",
)

http_file(
    name = "debian_deb_gxx_6",
    sha256 = "9fd0d72788ffff4a71ad4236731626c9074afc4a10990b58fe970072321bc26d",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/g++-6_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_grep",
    sha256 = "0704a54bfa9fb8e1045704cd7ce6f2f652a2cef9857fdd39cef97fd97c6d4d01",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/grep/grep_2.27-2_amd64.deb",
)

http_file(
    name = "debian_deb_libacl1",
    sha256 = "f5057ce95c6a6cf086253aa894f1d1c9bb3c13227224d389d8631190930a236d",
    url = "http://ftp.nl.debian.org/debian/pool/main/a/acl/libacl1_2.2.52-3+b1_amd64.deb",
)

http_file(
    name = "debian_deb_libattr1",
    sha256 = "b6c67690972c224643d85aed608f448c9c3e531fa13da3c3c1f17743c66015a4",
    url = "http://ftp.nl.debian.org/debian/pool/main/a/attr/libattr1_2.4.47-2+b2_amd64.deb",
)

http_file(
    name = "debian_deb_libc6",
    sha256 = "e57b3e24ea79fcdb46549d4ed2b95bb9657f21bba60ed5d9136d5b7112500084",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/glibc/libc6_2.24-11+deb9u3_amd64.deb",
)

http_file(
    name = "debian_deb_libc6_dev",
    sha256 = "8bdebd7bc1fc4138e0181821a1fe1fb576cbae241f03a31ab3c6cfc3a9875dc6",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/glibc/libc6-dev_2.24-11+deb9u3_amd64.deb",
)

http_file(
    name = "debian_deb_libgcc1",
    sha256 = "423a6541ee7ade69967c99492e267e724fd4675de53310861af5d1a1d249c4bf",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/libgcc1_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_libgcc_6_dev",
    sha256 = "fbaa19b872bee99a443319da415ae2de346d72d15b12dc3d0a4c3607b154b884",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/libgcc-6-dev_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_libgmp10",
    sha256 = "4a5ef027aae7d20060899e396113c55906d883d39675d9e9990bcace1acba0d1",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gmp/libgmp10_6.1.2+dfsg-1_amd64.deb",
)

http_file(
    name = "debian_deb_libisl15",
    sha256 = "7f0a81e458df5e9648252bf3a76ffd57f366a0ddcab5290a9c3bb5bc0c79e513",
    url = "http://ftp.nl.debian.org/debian/pool/main/i/isl/libisl15_0.18-1_amd64.deb",
)

http_file(
    name = "debian_deb_liblzma5",
    sha256 = "0dd0d94608fb054c7dca567692eab847d7f1dfd521ebe9c12bf7f6f64c95b006",
    url = "http://ftp.nl.debian.org/debian/pool/main/x/xz-utils/liblzma5_5.2.2-1.2+b1_amd64.deb",
)

http_file(
    name = "debian_deb_libmpc3",
    sha256 = "99b2bd2c8618494116bef1d13d0525fe2885be46e2441a4697afd7ec93efb431",
    url = "http://ftp.nl.debian.org/debian/pool/main/m/mpclib3/libmpc3_1.0.3-1+b2_amd64.deb",
)

http_file(
    name = "debian_deb_libmpfr4",
    sha256 = "95730a4709b898ffaf677f9b3ab6f6ebef5a96866589a8cf5f775448b3413a98",
    url = "http://ftp.nl.debian.org/debian/pool/main/m/mpfr4/libmpfr4_3.1.5-1_amd64.deb",
)

http_file(
    name = "debian_deb_libpcre3",
    sha256 = "d9a04a344b76190f4b14f2d3fc42b03b4db67efaa459d63549fcfd578935b11d",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/pcre3/libpcre3_8.39-3_amd64.deb",
)

http_file(
    name = "debian_deb_libperl524",
    sha256 = "e010ab8e7178c2271033aa199f925f1c2fd46e879d222462eaad35d1f7eaedea",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/perl/libperl5.24_5.24.1-3+deb9u4_amd64.deb",
)

http_file(
    name = "debian_deb_libpython27_minimal",
    sha256 = "06a6e0dfd5b41e503171ebc7083802a169a07a0c2aadca34a72afcf175f42dad",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/python2.7/libpython2.7-minimal_2.7.13-2+deb9u2_amd64.deb",
)

http_file(
    name = "debian_deb_libpython27_stdlib",
    sha256 = "3d7bdcf90b8766a2052f00ecaef9e4ef0348afba0c2a6693f6182f1925ac29f5",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/python2.7/libpython2.7-stdlib_2.7.13-2+deb9u2_amd64.deb",
)

http_file(
    name = "debian_deb_libselinux1",
    sha256 = "2d70d0f68783b14f812690cde1f1fcaade8befc6882f712fb7545bc86a207be0",
    url = "http://ftp.nl.debian.org/debian/pool/main/libs/libselinux/libselinux1_2.6-3+b3_amd64.deb",
)

http_file(
    name = "debian_deb_libstdcxx6",
    sha256 = "d05373fbbb0d2c538fa176dfe71d1fa7983c58d35a7a456263ca87e8e0d45030",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/libstdc++6_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_libstdcxx_6_dev",
    sha256 = "b13ce454a53108895efc7db8e1d99bd56d5e884ccbe174951586f12849fe82af",
    url = "http://ftp.nl.debian.org/debian/pool/main/g/gcc-6/libstdc++-6-dev_6.3.0-18+deb9u1_amd64.deb",
)

http_file(
    name = "debian_deb_libtinfo5",
    sha256 = "1d249a3193568b5ef785ad8993b9ba6d6fdca0eb359204c2355532b82d25e9f5",
    url = "http://ftp.nl.debian.org/debian/pool/main/n/ncurses/libtinfo5_6.0+20161126-1+deb9u2_amd64.deb",
)

http_file(
    name = "debian_deb_linux_libc_dev",
    sha256 = "2ecdb478eedc1206c2127d9ad646642347bd1f3947fdd64f1ae5c3cf45374a57",
    url = "http://ftp.nl.debian.org/debian/pool/main/l/linux/linux-libc-dev_4.9.110-1_amd64.deb",
)

http_file(
    name = "debian_deb_perl_base",
    sha256 = "914985af488a14268b911de8b06e082165f362e3d3c6a52581aa2619d557e1ea",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/perl/perl-base_5.24.1-3+deb9u4_amd64.deb",
)

http_file(
    name = "debian_deb_perl_modules_base",
    sha256 = "97ef07235d452887148df4791b24d50af224bebd47e90970d3b26eead718c330",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/perl/perl-modules-5.24_5.24.1-3+deb9u4_all.deb",
)

http_file(
    name = "debian_deb_python27_minimal",
    sha256 = "6f9769d212e1953432e101f0e5874182624204cfa61a8b322320b2c1d726193e",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/python2.7/python2.7-minimal_2.7.13-2+deb9u2_amd64.deb",
)

http_file(
    name = "debian_deb_python_minimal",
    sha256 = "425f1e6b2e1047a208b2e7c334455b8db2d0c03ea1ca761c4f53893a244c65c9",
    url = "http://ftp.nl.debian.org/debian/pool/main/p/python-defaults/python-minimal_2.7.13-2_amd64.deb",
)

http_file(
    name = "debian_deb_sed",
    sha256 = "0b241948d78b5b61755f29515bd7c3d60f8119f54176d9a7c454a3dd7b7a7b09",
    url = "http://ftp.nl.debian.org/debian/pool/main/s/sed/sed_4.4-1_amd64.deb",
)

http_file(
    name = "debian_deb_xz_utils",
    sha256 = "6d07d82ab8d58004f3bbe2ca82d1e812c94f84c297dce7d9a2d3bb7552cf0b57",
    url = "http://ftp.nl.debian.org/debian/pool/main/x/xz-utils/xz-utils_5.2.2-1.2+b1_amd64.deb",
)

http_file(
    name = "debian_deb_zlib1g",
    sha256 = "b5fe79041db0a5ed4c8297f7c1c2ec0c344be6eea0a7d0977a0f2a1dceea2ff3",
    url = "http://ftp.nl.debian.org/debian/pool/main/z/zlib/zlib1g_1.2.8.dfsg-5_amd64.deb",
)

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "com_github_aws_aws_sdk_go",
    commit = "bc3f534c19ffdf835e524e11f0f825b3eaf541c3",
    importpath = "github.com/aws/aws-sdk-go",
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
