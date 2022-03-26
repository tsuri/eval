FROM debian:buster-slim

RUN apt-get install wget build-essential python3
RUN wget https://github.com/bazelbuild/bazelisk/releases/download/v1.11.0/bazelisk-linux-amd64
RUN mv bazelisk-linux-amd64 /bin/bazel
RUN bazel build //test
