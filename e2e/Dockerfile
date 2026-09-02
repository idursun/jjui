# Docker builds the Linux test environment and runs the Go tests through the
# compose command. The Ghostty revision is pinned to the revision used by the
# current go.mitchellh.com/libghostty binding.
FROM golang:1.26-bookworm

ARG GHOSTTY_COMMIT=2dd79f3bc6af649e68422b08e21ad0300fd8b391
ARG ZIG_VERSION=0.16.0
ARG JJ_VERSION=0.44.0

ENV CGO_ENABLED=1
ENV PKG_CONFIG_PATH=/opt/libghostty-vt/share/pkgconfig

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
        ca-certificates \
        curl \
        git \
        pkg-config \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN case "$(dpkg --print-architecture)" in \
        amd64) zig_arch=x86_64 ;; \
        arm64) zig_arch=aarch64 ;; \
        *) echo "unsupported Docker architecture" >&2; exit 1 ;; \
    esac \
    && curl --fail --location --silent --show-error \
        "https://ziglang.org/download/${ZIG_VERSION}/zig-${zig_arch}-linux-${ZIG_VERSION}.tar.xz" \
        --output /tmp/zig.tar.xz \
    && tar -xJf /tmp/zig.tar.xz -C /opt \
    && ln -s "/opt/zig-${zig_arch}-linux-${ZIG_VERSION}/zig" /usr/local/bin/zig \
    && ln -s "/opt/zig-${zig_arch}-linux-${ZIG_VERSION}/lib" /usr/local/lib/zig \
    && rm /tmp/zig.tar.xz

RUN case "$(dpkg --print-architecture)" in \
        amd64) jj_arch=x86_64 ;; \
        arm64) jj_arch=aarch64 ;; \
        *) echo "unsupported Docker architecture" >&2; exit 1 ;; \
    esac \
    && mkdir /tmp/jj \
    && curl --fail --location --silent --show-error \
        "https://github.com/jj-vcs/jj/releases/download/v${JJ_VERSION}/jj-v${JJ_VERSION}-${jj_arch}-unknown-linux-musl.tar.gz" \
        --output /tmp/jj.tar.gz \
    && tar -xzf /tmp/jj.tar.gz -C /tmp/jj \
    && install /tmp/jj/jj /usr/local/bin/jj \
    && rm -rf /tmp/jj /tmp/jj.tar.gz

# This is the libghostty-vt source checkout. The library itself is built by
# Zig; the generated pkg-config file is consumed by cgo below.
RUN git init /opt/ghostty \
    && git -C /opt/ghostty remote add origin https://github.com/ghostty-org/ghostty.git \
    && git -C /opt/ghostty fetch --depth 1 origin "${GHOSTTY_COMMIT}" \
    && git -C /opt/ghostty checkout --detach FETCH_HEAD \
    && cd /opt/ghostty \
    && zig build -Demit-lib-vt --prefix /opt/libghostty-vt

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .

CMD ["go", "test", "./..."]
