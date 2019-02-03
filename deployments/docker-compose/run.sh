#!/usr/bin/env bash

# Self-contained Buildbarn setup that can be run using Docker Compose.
#
# Place the following options in ~/.bazelrc, so that you can let Bazel
# use it by running 'bazel build --config=bbb-debian8 <targets>' or
# 'bazel build --config=bbb-ubuntu16-04 <targets>'.
#
# # From bazel-toolchains/configs/debian8_clang/0.4.0/toolchain.bazelrc.
# build:bbb-debian8 --action_env=BAZEL_DO_NOT_DETECT_CPP_TOOLCHAIN=1
# build:bbb-debian8 --crosstool_top=@bazel_toolchains//configs/debian8_clang/0.4.0/bazel_0.20.0/default:toolchain
# build:bbb-debian8 --extra_execution_platforms=@bazel_toolchains//configs/debian8_clang/0.4.0:rbe_debian8
# build:bbb-debian8 --extra_toolchains=@bazel_toolchains//configs/debian8_clang/0.4.0/bazel_0.20.0/cpp:cc-toolchain-clang-x86_64-default
# build:bbb-debian8 --host_javabase=@bazel_toolchains//configs/debian8_clang/0.4.0:jdk8
# build:bbb-debian8 --host_java_toolchain=@bazel_tools//tools/jdk:toolchain_hostjdk8
# build:bbb-debian8 --host_platform=@bazel_toolchains//configs/debian8_clang/0.4.0:rbe_debian8
# build:bbb-debian8 --javabase=@bazel_toolchains//configs/debian8_clang/0.4.0:jdk8
# build:bbb-debian8 --java_toolchain=@bazel_tools//tools/jdk:toolchain_hostjdk8
# build:bbb-debian8 --platforms=@bazel_toolchains//configs/debian8_clang/0.4.0:rbe_debian8
# # Specific to the setup.
# build:bbb-debian8 --action_env=PATH=/bin:/usr/bin
# build:bbb-debian8 --cpu=k8
# build:bbb-debian8 --experimental_strict_action_env
# build:bbb-debian8 --genrule_strategy=remote
# build:bbb-debian8 --host_cpu=k8
# build:bbb-debian8 --jobs=8
# build:bbb-debian8 --remote_executor=localhost:8980
# build:bbb-debian8 --remote_instance_name=debian8
# build:bbb-debian8 --spawn_strategy=remote
# build:bbb-debian8 --strategy=Closure=remote
# build:bbb-debian8 --strategy=Javac=remote
#
# # From bazel-toolchains/configs/ubuntu16_04_clang/1.1/toolchain.bazelrc
# build:bbb-ubuntu16-04 --action_env=BAZEL_DO_NOT_DETECT_CPP_TOOLCHAIN=1
# build:bbb-ubuntu16-04 --crosstool_top=@bazel_toolchains//configs/ubuntu16_04_clang/1.1/bazel_0.20.0/default:toolchain
# build:bbb-ubuntu16-04 --extra_execution_platforms=@bazel_toolchains//configs/ubuntu16_04_clang/1.1:rbe_ubuntu1604
# build:bbb-ubuntu16-04 --extra_toolchains=@bazel_toolchains//configs/ubuntu16_04_clang/1.1/bazel_0.20.0/cpp:cc-toolchain-clang-x86_64-default
# build:bbb-ubuntu16-04 --host_javabase=@bazel_toolchains//configs/ubuntu16_04_clang/1.1:jdk8
# build:bbb-ubuntu16-04 --host_java_toolchain=@bazel_tools//tools/jdk:toolchain_hostjdk8
# build:bbb-ubuntu16-04 --host_platform=@bazel_toolchains//configs/ubuntu16_04_clang/1.1:rbe_ubuntu1604
# build:bbb-ubuntu16-04 --javabase=@bazel_toolchains//configs/ubuntu16_04_clang/1.1:jdk8
# build:bbb-ubuntu16-04 --java_toolchain=@bazel_tools//tools/jdk:toolchain_hostjdk8
# build:bbb-ubuntu16-04 --platforms=@bazel_toolchains//configs/ubuntu16_04_clang/1.1:rbe_ubuntu1604
# # Specific to the setup.
# build:bbb-ubuntu16-04 --action_env=PATH=/bin:/usr/bin
# build:bbb-ubuntu16-04 --cpu=k8
# build:bbb-ubuntu16-04 --experimental_strict_action_env
# build:bbb-ubuntu16-04 --genrule_strategy=remote
# build:bbb-ubuntu16-04 --host_cpu=k8
# build:bbb-ubuntu16-04 --jobs=8
# build:bbb-ubuntu16-04 --remote_executor=localhost:8980
# build:bbb-ubuntu16-04 --remote_instance_name=ubuntu16-04
# build:bbb-ubuntu16-04 --spawn_strategy=remote
# build:bbb-ubuntu16-04 --strategy=Closure=remote
# build:bbb-ubuntu16-04 --strategy=Javac=remote

set -eux

for component in frontend scheduler browser worker; do
  bazel run "//cmd/bbb_${component}:bbb_${component}_container"
done
bazel run "//cmd/bbb_runner:bbb_runner_debian8_container"
bazel run "//cmd/bbb_runner:bbb_runner_ubuntu16_04_container"
bazel run "@com_github_buildbarn_bb_storage//cmd/bb_storage:bb_storage_container"

for worker in worker-debian8 worker-ubuntu16-04; do
  rm -rf "${worker}"
  mkdir -m 0777 "${worker}" "${worker}/build"
  mkdir -m 0700 "${worker}/cache"
done

exec docker-compose up
