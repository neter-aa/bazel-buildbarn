#!/usr/bin/env bash

# Self-contained Buildbarn setup that can be run using Docker Compose.
#
# Place the following options in ~/.bazelrc, so that you can let Bazel
# use it by running 'bazel build --config=bbb-docker-compose <targets>'.
#
# # From bazel-toolchains/configs/debian8_clang/0.4.0/toolchain.bazelrc.
# build:bbb-docker-compose --action_env=BAZEL_DO_NOT_DETECT_CPP_TOOLCHAIN=1
# build:bbb-docker-compose --crosstool_top=@bazel_toolchains//configs/debian8_clang/0.4.0/bazel_0.20.0/default:toolchain
# build:bbb-docker-compose --extra_execution_platforms=@bazel_toolchains//configs/debian8_clang/0.4.0:rbe_debian8
# build:bbb-docker-compose --extra_toolchains=@bazel_toolchains//configs/debian8_clang/0.4.0/bazel_0.20.0/cpp:cc-toolchain-clang-x86_64-default
# build:bbb-docker-compose --host_javabase=@bazel_toolchains//configs/debian8_clang/0.4.0:jdk8
# build:bbb-docker-compose --host_java_toolchain=@bazel_tools//tools/jdk:toolchain_hostjdk8
# build:bbb-docker-compose --host_platform=@bazel_toolchains//configs/debian8_clang/0.4.0:rbe_debian8
# build:bbb-docker-compose --javabase=@bazel_toolchains//configs/debian8_clang/0.4.0:jdk8
# build:bbb-docker-compose --java_toolchain=@bazel_tools//tools/jdk:toolchain_hostjdk8
# build:bbb-docker-compose --platforms=@bazel_toolchains//configs/debian8_clang/0.4.0:rbe_debian8
# # Specific to the setup.
# build:bbb-docker-compose --action_env=PATH=/bin:/usr/bin
# build:bbb-docker-compose --cpu=k8
# build:bbb-docker-compose --experimental_strict_action_env
# build:bbb-docker-compose --genrule_strategy=remote
# build:bbb-docker-compose --host_cpu=k8
# build:bbb-docker-compose --jobs=8
# build:bbb-docker-compose --remote_executor=localhost:8980
# build:bbb-docker-compose --remote_instance_name=local
# build:bbb-docker-compose --spawn_strategy=remote
# build:bbb-docker-compose --strategy=Closure=remote
# build:bbb-docker-compose --strategy=Javac=remote

set -eux

#for component in frontend scheduler storage browser worker runner; do
#  bazel run "//cmd/bbb_${component}:bbb_${component}_container"
#done

rm -rf worker
mkdir -m 0777 worker worker/build
mkdir -m 0700 worker/cache

exec docker-compose up
