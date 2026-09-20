# Prevenir el warning de BuildKit sobre BASE_IMAGE
ARG BASE_IMAGE=golang:1.26.5-bookworm
FROM ${BASE_IMAGE}

# Instalar git y utilidades básicas
RUN apt-get update && apt-get install -y git openssh-client sudo && rm -rf /var/lib/apt/lists/*

# Crear un usuario no-root con UID 1000 si no existe
RUN id -u 1000 >/dev/null 2>&1 || useradd -m -u 1000 -s /bin/bash vscode \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers