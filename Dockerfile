# Building stage
FROM kairen/golang-dep:1.11-alpine AS build-env
LABEL maintainer="Kyle Bai <k2r2.bai@gmail.com>"

ENV GOPATH "/go"
ENV PROJECT_PATH "$GOPATH/src/github.com/kairen/line-bot-operator"

COPY . $PROJECT_PATH
RUN cd $PROJECT_PATH && \
  make dep && \
  make && mv out/controller /tmp/controller

# Running stage
FROM alpine:3.7
COPY --from=build-env /tmp/controller /bin/controller
ENTRYPOINT ["controller"]
