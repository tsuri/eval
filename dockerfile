FROM debian:buster-slim AS builder

RUN apt-get update
RUN apt-get install --yes wget build-essential python3
RUN wget -q https://releases.bazel.build/5.1.0/release/bazel-5.1.0-linux-x86_64 -O /usr/bin/bazel
RUN chmod +x /usr/bin/bazel
COPY . /eval
WORKDIR /eval
RUN echo $PWD
RUN ls
RUN /usr/bin/bazel build //test

FROM debian:buster-slim
COPY --from=builder /eval/bazel-bin/test /app
RUN ls /app
ENTRYPOINT /app/test