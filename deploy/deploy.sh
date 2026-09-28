#!/usr/bin/env bash
# Сборка и выкладка сайта на VPS вручную (обычно это делает GitHub Actions).
#   ./deploy/deploy.sh          — обычный деплой
#   ./deploy/deploy.sh --setup  — настройка сервера: пользователи, папки, служба, nginx, деплой
# SSH-хост берётся из DEPLOY_HOST (по умолчанию alias "arweb" из ~/.ssh/config),
# у пользователя должен быть sudo.
set -euo pipefail

HOST="${DEPLOY_HOST:-arweb}"
REL=/tmp/arweb-release
cd "$(dirname "$0")/.."

echo "→ сборка под Linux (amd64)"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o build/app .

echo "→ загрузка на $HOST"
ssh "$HOST" "mkdir -p $REL"
rsync -az --delete build/app templates static deploy "$HOST:$REL/"

ssh "$HOST" "sudo bash -s -- $REL ${1:-}" <<'REMOTE'
set -euo pipefail
REL="$1"
APP_DIR=/opt/ar-web-studio

if [ "${2:-}" = "--setup" ]; then
  id arweb >/dev/null 2>&1 || useradd --system --home "$APP_DIR" --shell /usr/sbin/nologin arweb
  mkdir -p "$APP_DIR/data"
  chown arweb:arweb "$APP_DIR" "$APP_DIR/data"
  chmod 755 "$APP_DIR"

  if [ ! -f /etc/ar-web-studio.env ]; then
    cat > /etc/ar-web-studio.env <<'ENV'
PORT=8080
GIN_MODE=release
# nginx на этом же сервере: доверяем X-Forwarded-For только от него
TRUSTED_PROXIES=127.0.0.1/32
# Заявки в Telegram: впишите значения и выполните sudo systemctl restart ar-web-studio
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=
ENV
    chmod 600 /etc/ar-web-studio.env
  fi

  # Пользователь для GitHub Actions: может только загрузить релиз и запустить установщик
  id deploy >/dev/null 2>&1 || useradd --create-home --shell /bin/bash deploy
  install -d -m 700 -o deploy -g deploy /home/deploy/.ssh /home/deploy/release
  touch /home/deploy/.ssh/authorized_keys
  chown deploy:deploy /home/deploy/.ssh/authorized_keys && chmod 600 /home/deploy/.ssh/authorized_keys
  # "" в конце запрещает передавать аргументы: только путь релиза по умолчанию
  echo 'deploy ALL=(root) NOPASSWD: /usr/local/bin/arweb-install ""' > /etc/sudoers.d/arweb-deploy
  chmod 440 /etc/sudoers.d/arweb-deploy
  visudo -cf /etc/sudoers.d/arweb-deploy >/dev/null

  install -m 644 "$REL/deploy/ar-web-studio.service" /etc/systemd/system/ar-web-studio.service
  if [ ! -e /etc/nginx/sites-available/ar-web-studio ]; then
    # после выпуска сертификата certbot дописывает этот файл, поэтому не перезаписываем его
    install -m 644 "$REL/deploy/nginx.conf" /etc/nginx/sites-available/ar-web-studio
  fi
  ln -sf /etc/nginx/sites-available/ar-web-studio /etc/nginx/sites-enabled/ar-web-studio
  rm -f /etc/nginx/sites-enabled/default
  systemctl daemon-reload
  systemctl enable ar-web-studio >/dev/null
  nginx -t -q && systemctl reload nginx
fi

install -m 755 -o root -g root "$REL/deploy/arweb-install" /usr/local/bin/arweb-install
/usr/local/bin/arweb-install "$REL"
rm -rf "$REL"
REMOTE
