#!/bin/bash

# Read input parameters from config.json
if [ -f "config.json" ]; then
    NETBOX_USERNAME=$(jq -r '.netbox_username' config.json)
    NETBOX_EMAIL=$(jq -r '.netbox_email' config.json)
    NETBOX_PASSWORD=$(jq -r '.netbox_pass' config.json)
else
    echo "config.json file not found."
    exit 1
fi

# Check if netbox-docker directory exists
if [ ! -d "netbox-docker" ]; then
    # Clone the NetBox Docker repository
    git clone -b release https://github.com/netbox-community/netbox-docker.git

    # Change to the netbox-docker directory
    cd netbox-docker

    # Create a docker-compose.override.yml file
    cat <<EOF > docker-compose.override.yml
version: '3.4'
services:
  netbox:
    ports:
    - 8000:8080
EOF

    # Pull the Docker images
    docker-compose pull
else
    echo "netbox-docker directory already exists."
    cd netbox-docker

    # Stop and remove all containers
    docker-compose down

    # Remove the superuser if it exists
    if docker-compose exec netbox /opt/netbox/netbox/manage.py shell -c "from django.contrib.auth.models import User; exit(1 if User.objects.filter(username='${NETBOX_USERNAME}').exists() else 0)"; then
        docker-compose exec netbox /opt/netbox/netbox/manage.py shell -c "from django.contrib.auth.models import User; User.objects.filter(username='${NETBOX_USERNAME}').delete()"
    fi
fi

# Start the NetBox containers
docker-compose up -d

# Check if NetBox containers are running
if ! docker-compose ps | grep -E "netbox_netbox_1.*Up" >/dev/null && [ "$(docker-compose ps | grep "netbox" | wc -l)" -ne 6 ]; then
    echo "Not all NetBox containers are running, or the number of containers is not as expected."
    exit 1
fi

if docker-compose exec netbox /opt/netbox/netbox/manage.py shell -c "from django.contrib.auth.models import User; exit(1 if User.objects.filter(username='${NETBOX_USERNAME}').exists() else 0)"; then
    if [ -z "$NETBOX_PASSWORD" ]; then
        read -s -p "Enter password for superuser: " NETBOX_PASSWORD
    fi
    expect -c "
    spawn docker-compose exec netbox /opt/netbox/netbox/manage.py createsuperuser --username $NETBOX_USERNAME --email $NETBOX_EMAIL
    expect \"Password:\"
    send \"$NETBOX_PASSWORD\n\"
    expect \"Password (again):\"
    send \"$NETBOX_PASSWORD\n\"
    interact
    "
fi
docker-compose up -d
sleep 15

