#!/bin/bash
# Install Hologram Server.

cat <<EOF > /etc/apt/apt.conf.d/90forceyes
APT::Get::Assume-Yes "true";
APT::Get::force-yes "true";
EOF


# Download the dependencies for this program.
apt-get update >> /tmp/init-script-log 2>&1
apt-get install python-pip curl jq >> /tmp/init-script-log 2>&1
pip install awscli >> /tmp/init-script-log 2>&1

# We need to know what region we're in.
INSTANCE_ID=`curl -sL http://169.254.169.254/latest/meta-data/instance-id`
AZ=`curl -sL http://169.254.169.254/latest/meta-data/placement/availability-zone`
REGION=${AZ%?}
export PATH=$PATH:/usr/local/bin

# Download APT s3 method.
aws s3 cp s3://adroll-hologram/support/s3.apt /usr/lib/apt/methods/s3
chmod a+x /usr/lib/apt/methods/s3

# Install my GPG key that packages are signed with.
# This allows us to have verified packages up in S3.
aws s3 cp s3://adroll-hologram/support/packages.gpg - | apt-key add - >> /tmp/init-script-log 2>&1

# Add sources.list.d entry for s3 repo.
echo "deb s3://s3-$REGION.amazonaws.com/adroll-hologram/repo stable main" > /etc/apt/sources.list.d/hologram.list
apt-get update >> /tmp/init-script-log 2>&1

# Download Hologram Package.
apt-get install hologram-server >> /tmp/init-script-log 2>&1

# Download and install Datadog agent.
DD_API_KEY=5b06dcdb5b36c5494c2c7f48d4eb8ea8 bash -c "$(curl -L https://raw.githubusercontent.com/DataDog/dd-agent/master/packaging/datadog-agent/source/install_agent.sh)"
update-rc.d datadog-agent defaults

# Set "online" tag to true.
aws ec2 create-tags --resources $INSTANCE_ID --tags Key=online,Value=true --region $REGION >> /tmp/init-script-log 2>&1

