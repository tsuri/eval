FROM registry.other.net:5000/eval/base-build AS builder
#FROM debian:buster-slim AS builder

# RUN apt-get update
# RUN apt-get install --yes wget build-essential python3
# RUN wget -q https://releases.bazel.build/5.1.0/release/bazel-5.1.0-linux-x86_64 -O /usr/bin/bazel
# RUN chmod +x /usr/bin/bazel
ARG TARGETS
COPY . /eval
WORKDIR /eval
RUN echo $PWD
RUN echo ${TARGETS}

#RUN /usr/bin/bazel build //test:test  //test:runner //test:sub //test:another
RUN /usr/bin/bazel build ${TARGETS}
RUN tar -chf /eval/bazel-out.tar -C /eval/bazel-bin .

FROM debian:buster

COPY --from=builder /eval/bazel-out.tar .
RUN tar -tf bazel-out.tar

#COPY --from=builder /eval/bazel-bin/     /app/
RUN ls -lRL /app
ENTRYPOINT /app/test/runner_/runner
