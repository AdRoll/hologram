#!/bin/sh

# If there's a current config file in place, save it
cp /etc/hologram/agent.json /etc/hologram/agent.json.save || exit 0
