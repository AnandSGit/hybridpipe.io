#!/bin/bash

# ------------------------------------------------------------------------------
# All the functions for this scripts should be defined here from here. Can be in any order
# ------------------------------------------------------------------------------

# Help - Helper Function
Help()
{
	echo "This script will install Kafka in this BASH environment (macOS)"
	echo "Kafka and Zookeeper versions are hardcoded as part of script file itself."
	echo "User need to pass the Broker as the only argument for this install script"
	echo "All IP Address and versions details are inside the script. Please change based on requirement"
	echo "--broker <BROKERID> as integer"
}

# Pour_Java - Installs Java
Pour_Java()
{
	add-apt-repository -y ppa:webupd8team/java
	apt-get -y update
	echo debconf shared/accepted-oracle-license-v1-1 select true | sudo debconf-set-selections
	echo debconf shared/accepted-oracle-license-v1-1 seen true | sudo debconf-set-selections
	apt-get -y install oracle-java7-installer
}
