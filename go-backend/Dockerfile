FROM golang:1.21 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
        make \
        git \
        && rm -rf /var/lib/apt/lists/*

COPY . /src
WORKDIR /src

RUN GOPROXY=https://goproxy.cn make build

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates  \
        netbase \
        curl \
        wget \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

# 安装grpcurl
RUN wget -O grpcurl.tar.gz https://github.com/fullstorydev/grpcurl/releases/download/v1.8.9/grpcurl_1.8.9_linux_x86_64.tar.gz \
    && tar -xzf grpcurl.tar.gz \
    && mv grpcurl /usr/local/bin/ \
    && rm grpcurl.tar.gz

COPY --from=builder /src/bin /app
COPY --from=builder /src/configs /app/configs

WORKDIR /app

EXPOSE 8000
EXPOSE 9000

CMD ["./go-backend", "-conf", "./configs"]