FROM debian:8.0

RUN apt-get update && apt-get install rsyslog ca-certificates -y
COPY objects/hologram-server.deb /tmp/hologram-server.deb
RUN dpkg -i /tmp/hologram-server.deb
ONBUILD COPY config.json /etc/hologram/server.json

COPY start.sh /start.sh

EXPOSE 3100

ENTRYPOINT ["/start.sh"]
