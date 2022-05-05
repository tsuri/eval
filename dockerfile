FROM registry.other.net:5000/eval/base-build AS builder

ARG TARGETS

COPY . /eval
WORKDIR /eval

RUN echo $PWD
RUN echo ${TARGETS}

RUN /usr/bin/bazel build ${TARGETS}
RUN tar -chf /eval/bazel-out.tar -C /eval/bazel-bin .

#--------------------------------------------------------------
FROM debian:buster

COPY --from=builder /eval/bazel-out.tar .
RUN mkdir /app
RUN tar -xf bazel-out.tar -C /app

#ENTRYPOINT /app/test/runner_/runner
