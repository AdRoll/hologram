#!/bin/sh
# Remove the previous version of Hologram.
launchctl unload -w /Library/LaunchDaemons/com.adroll.hologram-ip.plist
launchctl unload -w /Library/LaunchDaemons/com.adroll.hologram.plist
launchctl unload -w /Library/LaunchDaemons/com.adroll.hologram-me.plist
launchctl unload -w /Library/LaunchAgents/com.adroll.hologram-me.plist

if [ -f "/Library/LaunchDaemons/com.adroll.hologram-me.plist" ]; then
    rm /Library/LaunchDaemons/com.adroll.hologram-me.plist
fi

# Remove previous (old location) hologram binaries if they exist
if [ -f "/usr/bin/hologram-boot" ]; then
  rm /usr/bin/hologram-boot
fi

if [ -f "/usr/bin/hologram-agent" ]; then
  rm /usr/bin/hologram-agent
fi

if [ -f "/usr/bin/hologram-authorize" ]; then
  rm /usr/bin/hologram-authorize
fi

if [ -f "/usr/bin/hologram" ]; then
  rm /usr/bin/hologram
fi

# Copy our previous config file over the new one
if [ -f "/etc/hologram/agent.json.save" ]; then
    mv /etc/hologram/agent.json /etc/hologram/agent.json.pkgnew
    mv /etc/hologram/agent.json.save /etc/hologram/agent.json
fi

# Load the services
launchctl load -w /Library/LaunchDaemons/com.adroll.hologram-ip.plist
launchctl load -w /Library/LaunchDaemons/com.adroll.hologram.plist
launchctl load -w /Library/LaunchAgents/com.adroll.hologram-me.plist
