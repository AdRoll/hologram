#!/bin/sh
# Remove the previous version of Hologram.
launchctl unload -w /Library/LaunchDaemons/com.adroll.hologram.plist

launchctl load -w /Library/LaunchDaemons/com.adroll.hologram-ip.plist
launchctl load -w /Library/LaunchDaemons/com.adroll.hologram.plist
launchctl load -w /Library/LaunchAgents/com.adroll.hologram-me.plist

