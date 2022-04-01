FROM debian:buster-slim AS builder

RUN apt-get update
RUN apt-get install --yes wget build-essential python3
RUN wget -q https://releases.bazel.build/5.1.0/release/bazel-5.1.0-linux-x86_64 -O /usr/bin/bazel
RUN chmod +x /usr/bin/bazel
COPY . /eval
WORKDIR /eval
RUN echo $PWD
RUN ls
#RUN /usr/bin/bazel build //test:runner
RUN /usr/bin/bazel build //test:test  //test:runner

FROM debian:buster
#RUN apt-get update && apt-get install --yes python3
#FROM gcr.io/distroless/python3
COPY --from=builder /eval/bazel-bin/test     /app
ENTRYPOINT /app/runner_/runner