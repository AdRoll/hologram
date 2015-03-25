FROM golang:1.4-cross
# Example build:
# docker build -t hologram_build .
# docker run -v $(pwd):/go/src/github.com/AdRoll/hologram prueba3
# Artifacts will be ready at $GOPATH/src/github.com/AdRoll/hologram/artifacts

#TODO: Try to reduce size of image

RUN apt-get update && apt-get install -y ruby-dev gcc ruby cpio gcc libssl-dev libxml2-dev make g++ rsyslog
RUN gem install fpm --no-rdoc

WORKDIR /tmp
RUN git clone https://github.com/pote/gpm.git && cd gpm && ./configure && make install && rm -rf /tmp/gpm
RUN git clone https://github.com/pote/gvp.git && cd gvp && ./configure && make install && rm -rf /tmp/gvp
RUN wget http://xar.googlecode.com/files/xar-1.5.2.tar.gz && tar xf xar-1.5.2.tar.gz && cd xar-1.5.2 && ./configure && make && make install && rm -rf /tmp/xar-1.5.2
RUN git clone https://github.com/hogliux/bomutils.git && cd bomutils && make && make install && rm -rf /tmp/bomutils

COPY buildscripts/test_and_build.sh /tmp/test_and_build.sh

WORKDIR /go/src/github.com/AdRoll/hologram

VOLUME ["/go/src/github.com/AdRoll/hologram"]

ENTRYPOINT ["/tmp/test_and_build.sh"]
