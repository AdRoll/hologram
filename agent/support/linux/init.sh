#!/bin/bash
# Hologram agent for Ubuntu machines.
# chkconfig: 345 20 80
# description: Hologram agent.
# processname: hologram-agent

DAEMON_PATH="/usr/local/bin"

DAEMON="/usr/local/bin/hologram-agent"
DAEMONOPTS=""

NAME="hologram-agent"
DESC="AWS Credentials Agent"
PIDFILE=/var/run/$NAME.pid
SCRIPTNAME=/etc/init.d/$NAME

case "$1" in
start)
  printf "%-50s" "Starting $NAME..."
  cd $DAEMON_PATH
  # Make sure that the metadata interface is up.
  ip addr add 169.254.169.254/24 broadcast 169.254.169.255 dev lo:metadata
  ip link set dev lo:metadata up
  PID=`$DAEMON $DAEMONOPTS > /var/log/hologram.log 2>&1 & echo $!`
  #echo "Saving PID" $PID " to " $PIDFILE
        if [ -z $PID ]; then
            printf "%s\n" "Fail"
        else
            echo $PID > $PIDFILE
            printf "%s\n" "Ok"
        fi
;;
status)
        printf "%-50s" "Checking $NAME..."
        if [ -f $PIDFILE ]; then
            PID=`cat $PIDFILE`
            if [ -z "`ps axf | grep ${PID} | grep -v grep`" ]; then
                printf "%s\n" "Process dead but pidfile exists"
            else
                echo "Running"
            fi
        else
            printf "%s\n" "Service not running"
        fi
;;
stop)
        printf "%-50s" "Stopping $NAME"
            PID=`cat $PIDFILE`
            cd $DAEMON_PATH
        if [ -f $PIDFILE ]; then
            kill -TERM $PID
            printf "%s\n" "Ok"
            rm -f $PIDFILE
        else
            printf "%s\n" "pidfile not found"
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

