#!/bin/bash

function start_server() {
    local BUILD_TYPE="$1"
    if [[ "$BUILD_TYPE" == "docker" ]]; then
        docker load -i gemfast*.tar
        docker run -d --name gemfast-server -p 80:2020 -v /etc/gemfast:/etc/gemfast -v /var/gemfast:/var/gemfast -v /etc/machine-id:/etc/machine-id server:latest
        sleep 5
        docker ps
        docker logs gemfast-server
    else
        sudo dpkg -i gemfast*.deb
        sudo systemctl start gemfast
        sleep 10
        sudo systemctl status gemfast
        sleep 2
        sudo systemctl status caddy

        journalctl -u gemfast
        journalctl -u caddy
    fi
}

function restart_server() {
    local BUILD_TYPE="$1"
    if [[ "$BUILD_TYPE" == "docker" ]]; then
        docker restart gemfast-server
        sleep 10
        docker ps
        docker logs gemfast-server
    else
        sudo systemctl restart gemfast
        sleep 5
        sudo systemctl status gemfast
        sleep 2
        sudo systemctl status caddy
    fi
}