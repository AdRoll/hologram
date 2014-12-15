#!/bin/bash
# Wait for the network to come online before booting Hologram.
while true
do
  sleep 1
  if host hologram.internal.adroll.com; then
    echo "Booting Hologram."
    /usr/bin/hologram me
    break
  else
    echo "Not online."
  fi
done
