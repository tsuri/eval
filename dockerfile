FROM debian:buster-slim

RUN apt-get update
RUN apt-get install --yes wget build-essential python3
RUN wget -q https://releases.bazel.build/5.1.0/release/bazel-5.1.0-linux-x86_64 -O /usr/bin/bazel
RUN chmod +x /usr/bin/bazel
WORKDIR /eval
RUN echo $PWD
RUN cat /eval/WORKSPACE
RUN cat WORKSPACE
RUN ls
RUN cd /eval && /usr/bin/bazel build //test
