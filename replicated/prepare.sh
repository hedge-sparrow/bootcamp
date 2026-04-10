#!/bin/sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

IMAGE_TAG="sha-$(git -C "$REPO_ROOT" rev-parse --short HEAD)"
echo "Image tag: $IMAGE_TAG"

echo "Cleaning up old chart archives..."
rm -f "$SCRIPT_DIR"/*.tgz

echo "Packaging bootcamp Helm chart (appVersion: $IMAGE_TAG)..."
helm package "$REPO_ROOT/helm" --app-version "$IMAGE_TAG" --destination "$SCRIPT_DIR"

echo "Downloading cloudnative-pg chart..."
helm pull cloudnative-pg \
  --repo https://cloudnative-pg.github.io/charts \
  --version 0.22.0 \
  --destination "$SCRIPT_DIR"

echo "Downloading traefik chart..."
helm pull traefik \
  --repo https://helm.traefik.io/traefik \
  --version 33.0.0 \
  --destination "$SCRIPT_DIR"

echo "Updating builder image tag in helmchart.yaml..."
awk -v tag="$IMAGE_TAG" '
  /repository: ghcr\.io\/ashjones-replicated\/bootcamp/ { found=1 }
  found && /tag:/ { sub(/tag: "[^"]*"/, "tag: \"" tag "\""); found=0 }
  { print }
' "$SCRIPT_DIR/helmchart.yaml" > "$SCRIPT_DIR/helmchart.yaml.tmp" \
  && mv "$SCRIPT_DIR/helmchart.yaml.tmp" "$SCRIPT_DIR/helmchart.yaml"

echo "Done. Release artifacts in $SCRIPT_DIR:"
ls "$SCRIPT_DIR"/*.tgz
