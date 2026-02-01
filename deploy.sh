#!/bin/bash
set -euo pipefail

# Deploy userscript-lab to remote SSH server
# Usage: ./deploy.sh

SSH_KEY="/Users/twesh/.ssh/scriptwright"
SSH_USER="alfa"
SSH_HOST="scriptwright"
SSH_TARGET="${SSH_USER}@${SSH_HOST}"
CONTAINER_NAME="userscript-lab"
IMAGE_NAME="userscript-lab:latest"
REMOTE_PORT="8787"
REMOTE_DIR="/home/${SSH_USER}/userscript-lab"

echo "==> Creating remote directory..."
ssh -i "${SSH_KEY}" "${SSH_TARGET}" "mkdir -p ${REMOTE_DIR}"

echo "==> Copying source code to remote server ${SSH_TARGET}..."
rsync -avz --exclude 'runs' --exclude '.git' --exclude 'node_modules' --exclude '.DS_Store' \
  -e "ssh -i ${SSH_KEY}" \
  ./ "${SSH_TARGET}:${REMOTE_DIR}/"

echo "==> Building container image on remote server..."
ssh -i "${SSH_KEY}" "${SSH_TARGET}" "cd ${REMOTE_DIR} && podman build -f Containerfile -t ${IMAGE_NAME} ."

echo "==> Stopping existing container (if any)..."
ssh -i "${SSH_KEY}" "${SSH_TARGET}" "podman stop ${CONTAINER_NAME} 2>/dev/null || true"
ssh -i "${SSH_KEY}" "${SSH_TARGET}" "podman rm ${CONTAINER_NAME} 2>/dev/null || true"

echo "==> Starting container on remote server..."
ssh -i "${SSH_KEY}" "${SSH_TARGET}" \
  "podman run -d --name ${CONTAINER_NAME} \
    --restart unless-stopped \
    -p ${REMOTE_PORT}:8787 \
    -v /home/${SSH_USER}/runs:/app/runs \
    -v /home/${SSH_USER}/extensions:/app/extensions \
    ${IMAGE_NAME}"

echo "==> Cleaning up local tarball..."
rm -f /tmp/userscript-lab.tar

echo ""
echo "✅ Deployment complete!"
echo "   Remote server: ${SSH_TARGET}"
echo "   Container: ${CONTAINER_NAME}"
echo "   Port: ${REMOTE_PORT}"
echo ""
echo "To access the UI, SSH tunnel or access directly:"
echo "  ssh -i ${SSH_KEY} -L 8787:localhost:8787 ${SSH_TARGET}"
echo "  Then open: http://localhost:8787/ui/"
echo ""
echo "To check logs:"
echo "  ssh -i ${SSH_KEY} ${SSH_TARGET} podman logs -f ${CONTAINER_NAME}"
echo ""
echo "To stop the service:"
echo "  ssh -i ${SSH_KEY} ${SSH_TARGET} podman stop ${CONTAINER_NAME}"
