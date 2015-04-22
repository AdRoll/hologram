#!/bin/bash
# Hologram server production deployment.
# chkconfig: 345 20 80
# description: Hologram server
# processname: hologram-server

set -e

DAEMON_PATH="/usr/local/bin"

DAEMON="/usr/local/bin/hologram-server"
DAEMONOPTS=""

NAME=hologram-server
DESC="AWS Credentials Server"
PIDFILE=/var/run/$NAME.pid
SCRIPTNAME=/etc/init.d/$NAME

case "$1" in
start)
  printf "%-50s" "Starting $NAME..."
  cd $DAEMON_PATH
  PID=`$DAEMON $DAEMONOPTS > /var/log/hologram.log 2>&1 & echo $!`
  RETCODE=$?
  #echo "Saving PID" $PID " to " $PIDFILE
        if [ -z $PID ]; then
            printf "%s\n" "Fail"
            exit $RETCODE
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
                exit 1
            else
                echo "Running"
                exit 0
            fi
        else
            printf "%s\n" "Service not running"
            exit 3
        fi
;;
stop)
        printf "%-50s" "Stopping $NAME"
        if [ -f $PIDFILE ]; then
            cd $DAEMON_PATH
            PID=`cat $PIDFILE`
            kill -TERM $PID
            RETCODE=$?
            if [ $RETCODE -eq 0 ]; then
                printf "%s\n" "Ok"
                rm -f $PIDFILE
            else
                printf "%s\n" "Error stopping service"
                exit $RETCODE
            fi
        else
            printf "%s\n" "pidfile not found"
            exit 1
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

