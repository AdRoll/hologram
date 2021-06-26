#!/bin/bash
# Hologram agent for Linux machines.
# chkconfig: 345 20 80
# description: Hologram agent.
# processname: hologram-agent

DAEMON_PATH=/usr/local/bin
NAME=hologram-agent
DAEMON=$DAEMON_PATH/$NAME
DAEMONOPTS=

DESC='AWS Credentials Agent'
PIDFILE=/var/run/$NAME.pid
SCRIPTNAME=/etc/init.d/$NAME

case $1 in

    start)
        printf '%-50s' "Starting $NAME..."
        cd "$DAEMON_PATH"
        # Make sure that the metadata interface is up.
        ip addr add 169.254.169.254/24 broadcast 169.254.169.255 dev lo:metadata
        ip link set dev lo:metadata up
        pid=$("$DAEMON" $DAEMONOPTS &> /var/log/hologram.log & echo $!)
        if [[ $pid ]]; then
            echo "$pid" > "$PIDFILE"
            printf '%s\n' Ok
        else
            printf '%s\n' Fail
        fi
    ;;

    status)
        printf '%-50s' "Checking $NAME..."
        if [ -f $PIDFILE ]; then
            pid=$(cat "$PIDFILE")
            if [ -z "$(ps axf | grep "$pid" | grep -v grep)" ]; then
                printf '%s\n' "Process dead but pidfile exists"
            else
                echo Running
            fi
        else
            printf '%s\n' 'Service not running'
        fi
    ;;

    stop)
        printf '%-50s' "Stopping $NAME"
        pid=$(cat "$PIDFILE")
        cd $DAEMON_PATH
        if [ -f $PIDFILE ]; then
            kill -TERM "$pid"
            printf '%s\n' Ok
            rm -f "$PIDFILE"
        else
            printf '%s\n' 'pidfile not found'
        fi
    ;;

    restart)
        $0 stop
        $0 start
    ;;

    *)
        echo "Usage: $0 {status|start|stop|restart}"
        exit 1

esac
