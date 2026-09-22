#!/usr/bin/env bash
set -euo pipefail

deploy_path="${1:?deploy path is required}"
service="${2:?compose service is required}"
container="${3:?container name is required}"
image_variable="${4:?image variable is required}"
image="${5:?image is required}"
healthcheck_url="${6:?health-check URL is required}"

if [[ "$deploy_path" != /* || "$deploy_path" == "/" ]]; then
  echo "Unsafe deploy path: $deploy_path" >&2
  exit 1
fi

if [[ ! "$service" =~ ^[a-z0-9_-]+$ || ! "$container" =~ ^[a-z0-9_-]+$ ]]; then
  echo "Invalid service or container name" >&2
  exit 1
fi

if [[ ! "$image_variable" =~ ^[A-Z0-9_]+$ ]]; then
  echo "Invalid image variable: $image_variable" >&2
  exit 1
fi

compose_file="${deploy_path}/compose.production.yaml"
if [[ ! -f "$compose_file" ]]; then
  echo "Compose file not found: $compose_file" >&2
  exit 1
fi

previous_image="$(docker inspect --format '{{.Config.Image}}' "$container" 2>/dev/null || true)"

cd "$deploy_path"
export "${image_variable}=${image}"
docker compose -f "$compose_file" pull "$service"
docker compose -f "$compose_file" up -d "$service"

healthy=false
for _ in {1..15}; do
  if curl --fail --silent --show-error --max-time 3 "$healthcheck_url" >/dev/null; then
    healthy=true
    break
  fi
  sleep 2
done

if [[ "$healthy" != true ]]; then
  echo "Health-check failed for ${service}. Rolling back." >&2
  if [[ -n "$previous_image" ]]; then
    export "${image_variable}=${previous_image}"
    docker compose -f "$compose_file" up -d "$service"
  fi
  exit 1
fi

echo "${service} is running image ${image}."
