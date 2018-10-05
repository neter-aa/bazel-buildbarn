# Bazel Buildbarn

Bazel Buildbarn is an implementation of a Bazel
[buildfarm](https://en.wikipedia.org/wiki/Compile_farm) written in the
Go programming language. The intent behind this implementation is that
it is fast and easy to scale. It consists of the following three
components:

- `bbb_frontend`: A service capable of processing RPCs from Bazel. It
  can store build input and serve cached build output and action results.
- `bbb_scheduler`: A service that receives requests from `bbb_frontend`
  to queue build actions that need to be run.
- `bbb_worker`: A service that runs build actions by fetching them from
  the `bbb_scheduler`.

The `bbb_frontend` and `bbb_worker` services can be replicated easily.
It is also possible to start multiple `bbb_scheduler` processes if
multiple build queues are desired (e.g., supporting multiple build
operating systems).

These processes depend on a central data store to cache their data.
Several storage backends are supported: [Redis](https://redis.io/),
[S3](https://aws.amazon.com/s3/) and [Bazel Remote](https://github.com/buchgr/bazel-remote/).
Multiple backends can be used in a single deployment to combine their
individual strengths. For example, Redis is efficient at storing small
objects, whereas S3 is better suited for large objects. Bazel Buildbarn
can be configured to partition objects in the Content Addressable
Storage across backends by size.

Below is a diagram of what a typical Bazel Buildbarn deployment may look
like. In this diagram, the arrows represent the direction in which
network connections are established.

<p align="center">
  <img src="https://github.com/EdSchouten/bazel-buildbarn/raw/master/doc/diagrams/bbb-overview.png" alt="Overview of a typical Bazel Buildbarn deployment"/>
</p>

One common use case for this implementation is to be run in Docker
containers on Kubernetes. In such environments it is
generally impossible to use [sandboxfs](https://github.com/bazelbuild/sandboxfs/),
meaning `bbb_worker` uses basic UNIX credentials management (privilege
separation) to provide a rudimentary form of sandboxing. The
`bbb_worker` daemon runs as user `root`, whereas the build action is run
as user `build`. Input files are only readable to the latter.

## Setting up Bazel Buildbarn

This repository contains publicly visible targets that build Docker
container images for the individual components:

    //cmd/bbb_worker:bbb_worker_container
    //cmd/bbb_scheduler:bbb_scheduler_container
    //cmd/bbb_frontend:bbb_frontend_container

You can add this repository to an existing workspace and use
[`container_push()`](https://github.com/bazelbuild/rules_docker#container_push-1)
rules to push these three container images to a container registry of
choice.

The `kubernetes/` directory in this repository contains example YAML
files that you may use to run Bazel Buildbarn on Kubernetes. Only YAML
files for Bazel Buildbarn itself are provided. Instructions on how to
set up dependencies, such as Redis and S3, are not included.

## Using Bazel Buildbarn

Bazel can be configured to perform remote execution against Bazel Buildbarn by
placing the following in `.bazelrc`:

    build:bbb-debian9 --host_cpu=k8 --cpu=k8 --crosstool_top=@bazel_buildbarn//toolchain:debian9 --experimental_strict_action_env --genrule_strategy=remote --remote_executor=address.of.your.buildbarn.deployment.here.com:8980 --remote_instance_name=debian9 --spawn_strategy=remote --strategy=Closure=remote --strategy=Javac=remote

In the configuration above, we assume that the container image for the
worker based on Debian 9 is used. For this image, we depend on a C/C++
compiler configuration for Debian 9 that is stored in this repository,
meaning that you will need to add a `bazel_buildbarn` repository
reference to your `WORKSPACE` file. Once added, you may perform remote
builds against Bazel Buildbarn by running the command below:

    bazel build --config=bbb-debian9 //...
