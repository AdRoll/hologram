FROM golang:1.4
# Example build:
# docker build -t hologram_build .
# docker run -v $(pwd):/go/src/github.com/AdRoll/hologram prueba3
# Artifacts will be ready at $GOPATH/src/github.com/AdRoll/hologram/artifacts

# Copied straight from the golang:1.4-cross Dockerfile, but reducing the number of platforms
RUN cd /usr/src/go/src \
    && set -ex \
    && for GOOS in darwin linux; do \
        GOOS=$GOOS ./make.bash --no-clean 2>&1; \
    done

#TODO: Try to reduce size of image

RUN apt-get update && apt-get install -y ruby-dev gcc ruby cpio gcc libssl-dev libxml2-dev make g++ rsyslog
RUN gem install fpm --no-rdoc

RUN cd /tmp && wget https://github.com/google/protobuf/releases/download/v2.6.1/protobuf-2.6.1.tar.gz && tar -zxvf protobuf-2.6.1.tar.gz && cd protobuf-2.6.1 && ./configure --prefix=/usr && make && make install

# Avoid using ssh to get the repos
#RUN git config --global url."https://github.com/".insteadOf "git@github.com:"

WORKDIR /tmp
# Get dependencies for building hologram
run go get github.com/jteeuwen/go-bindata/...
RUN git clone https://github.com/pote/gpm.git && cd gpm && ./configure && make install && rm -rf /tmp/gpm
RUN wget http://xar.googlecode.com/files/xar-1.5.2.tar.gz && tar xf xar-1.5.2.tar.gz && cd xar-1.5.2 && ./configure && make && make install && rm -rf /tmp/xar-1.5.2
RUN git clone https://github.com/hogliux/bomutils.git && cd bomutils && make && make install && rm -rf /tmp/bomutils




ENV HOLOGRAM_DIR /go/src/github.com/AdRoll/hologram
ENV BUILD_SCRIPTS ${HOLOGRAM_DIR}/buildscripts
ENV PATH ${BUILD_SCRIPTS}:$PATH
ENV BIN_DIR /go/bin
WORKDIR /go/src/github.com/AdRoll/hologram

VOLUME ["/go/src/github.com/AdRoll/hologram"]

ENTRYPOINT ["start.sh"]
