FROM debian:buster-slim

RUN apt-get update
RUN apt-get install --yes wget build-essential python3
RUN wget https://github.com/bazelbuild/bazelisk/releases/download/v1.11.0/bazelisk-linux-amd64
RUN mv bazelisk-linux-amd64 /bin/bazel
RUN chmod +x /bin/bazel
RUN /bin/bazel build //test
