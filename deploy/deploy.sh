#!/usr/bin/env bash
# Сборка и выкладка сайта на VPS.
#   ./deploy/deploy.sh          — обычный деплой
#   ./deploy/deploy.sh --setup  — первый запуск: пользователь, папки, служба, nginx
# SSH-хост берётся из переменной DEPLOY_HOST (по умолчанию alias "arweb" из ~/.ssh/config).
set -euo pipefail

HOST="${DEPLOY_HOST:-arweb}"
APP_DIR=/opt/ar-web-studio
cd "$(dirname "$0")/.."

echo "→ сборка под Linux (amd64)"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o build/app .

echo "→ загрузка на $HOST"
ssh "$HOST" "mkdir -p /tmp/arweb-release"
rsync -az --delete build/app templates static deploy "$HOST:/tmp/arweb-release/"

ssh "$HOST" "sudo bash -s -- ${1:-}" <<'REMOTE'
set -euo pipefail
APP_DIR=/opt/ar-web-studio
REL=/tmp/arweb-release

if [ "${1:-}" = "--setup" ]; then
  id arweb >/dev/null 2>&1 || useradd --system --home "$APP_DIR" --shell /usr/sbin/nologin arweb
  mkdir -p "$APP_DIR/data"
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
  install -m 644 "$REL/deploy/ar-web-studio.service" /etc/systemd/system/ar-web-studio.service
  install -m 644 "$REL/deploy/nginx.conf" /etc/nginx/sites-available/ar-web-studio
  ln -sf /etc/nginx/sites-available/ar-web-studio /etc/nginx/sites-enabled/ar-web-studio
  rm -f /etc/nginx/sites-enabled/default
  systemctl daemon-reload
  systemctl enable ar-web-studio >/dev/null
fi

# Код и шаблоны заменяем целиком, папку data/ с заявками не трогаем
install -m 755 "$REL/app" "$APP_DIR/app"
rm -rf "$APP_DIR/templates" "$APP_DIR/static"
cp -r "$REL/templates" "$REL/static" "$APP_DIR/"
chown -R arweb:arweb "$APP_DIR"
chmod 755 "$APP_DIR"

systemctl restart ar-web-studio
nginx -t -q && systemctl reload nginx

sleep 1
systemctl is-active --quiet ar-web-studio && echo "✓ ar-web-studio запущен" || { journalctl -u ar-web-studio -n 20 --no-pager; exit 1; }
rm -rf "$REL"
REMOTE
