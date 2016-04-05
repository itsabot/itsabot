#!/bin/sh

echo "This script requires superuser access to install golang, git, and/or postgres."
echo "You will be prompted for your password by sudo."

# clear any previous sudo permission
sudo -k
if [[ "$OSTYPE" == "linux-gnu" ]]; then

    # update your sources
    sudo apt-get update

    # install golang
    if [ $(dpkg-query -W -f='${Status}' golang 2>/dev/null | grep -c "ok installed") -eq 0 ];
    then
        sudo apt-get install -y golang
    fi

    # install git
    if [ $(dpkg-query -W -f='${Status}' git 2>/dev/null | grep -c "ok installed") -eq 0 ];
    then
        sudo apt-get install -y git
    fi

    # install postgresql
    if [ $(dpkg-query -W -f='${Status}' postgresql 2>/dev/null | grep -c "ok installed") -eq 0 ];
    then
        sudo apt-get install -y postgresql postgresql-contrib
        echo
        echo "You will now set the default password for the postgres user."
        echo "This will open a psql terminal, enter:"
        echo
        echo "\password postgres"
        echo
        echo "and follow instructions for setting postgres admin password."
        echo "Press Ctrl+D or type \\q to quit psql terminal"
        echo "START psql -------------------"
        sudo -u postgres psql postgres
        echo "END psql ---------------------"
        echo
    fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
    # This is the code for when we're on OS X
    /usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"

    brew update
    brew doctor
    brew install git
    brew install go --cross-compile-common
    brew install postgresql

    createuser -d postgres

    echo
    echo "You will now set the default password for the postgres user."
    echo "This will open a psql terminal, enter:"
    echo
    echo "\password postgres"
    echo
    echo "and follow instructions for setting postgres admin password."
    echo "Press Ctrl+D or type \\q to quit psql terminal"
    echo "START psql -------------------"
    psql postgres
    echo "END psql ---------------------"
    echo
    mkdir -p ~/Library/LaunchAgents

    ln -sfv /usr/local/opt/postgresql/*.plist ~/Library/LaunchAgents

    launchctl load ~/Library/LaunchAgents/homebrew.mxcl.postgresql.plist
fi

FILEPATH="$HOME/go"
printf "Enter a path for your GOPATH: ($FILEPATH) "
read tempPath
printf "\n"
[ -n "$tempPath" ] && FILEPATH=$tempPath

if [ ! -d "$FILEPATH" ]; then
    sudo mkdir $FILEPATH
fi

export GOPATH=$FILEPATH
export PATH=$PATH:$GOPATH/bin
touch $HOME/.bashrc
echo "# GOPATH is used to specify directories outside of \$GOROOT" >> $HOME/.bashrc
echo "# that contain the source for Go projects and their binaries." >> $HOME/.bashrc
echo "export GOPATH=$FILEPATH" >> $HOME/.bashrc
echo "# GOPATH/bin stores compiled go binaries" >> $HOME/.bashrc
echo "export PATH=\$PATH:$GOPATH/bin" >> $HOME/.bashrc
sudo chown -R $USER $GOPATH

stty -echo
printf "Please enter the postgresql password you just setup: "
read PASS
stty echo
printf "\n"

go get github.com/itsabot/abot
cd $GOPATH/src/github.com/itsabot/abot
cmd/setup.sh postgres:$PASS@127.0.0.1:5432
