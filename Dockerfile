FROM golang:1.27.1

RUN apt-get update && apt-get install -y --no-install-recommends \
    cpio \
    file \
    g++ \
    gcc \
    git \
    jq \
    libssl-dev \
    libxml2-dev \
    llvm-19 \
    make \
    rpm \
    rsyslog \
    ruby \
    ruby-dev \
    xar \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

RUN gem install --no-document fpm

WORKDIR /tmp
# mkbom, for the payload manifest in the macOS flat package.
RUN git clone --depth 1 https://github.com/hogliux/bomutils.git > /dev/null \
    && make -C bomutils install > /dev/null \
    && rm -rf /tmp/bomutils

# llvm-lipo combines the darwin amd64 and arm64 builds into a single universal
# binary; Apple's lipo is macOS-only, so this keeps the whole build in the
# container. Debian only ships the tool under its LLVM version, so alias it and
# keep the version pinned to the package name above.
RUN ln -s "$(ls /usr/bin/llvm-lipo-* | head -1)" /usr/local/bin/llvm-lipo

ENV HOLOGRAM_DIR=/go/src/github.com/AdRoll/hologram
ENV BUILD_SCRIPTS=${HOLOGRAM_DIR}/buildscripts
ENV PATH=${BUILD_SCRIPTS}:$PATH
ENV BIN_DIR=/go/bin

# The repo is bind-mounted at runtime and owned by the host user, which git
# refuses to touch by default.
RUN git config --global --add safe.directory ${HOLOGRAM_DIR}

COPY . /go/src/github.com/AdRoll/hologram
WORKDIR /go/src/github.com/AdRoll/hologram

VOLUME ["/go/src/github.com/AdRoll/hologram"]

ENTRYPOINT ["start.sh"]
