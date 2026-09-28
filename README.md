# AR Web Studio

Сайт-портфолио веб-студии: сайты для малого бизнеса в Алматы.
Стек: **Go + Gin** (сервер, шаблоны, API заявок), **HTML**, **CSS**, **JavaScript** без фреймворков.

## Запуск

```bash
go run .
```

Откройте http://localhost:8080

## Структура

| Путь | Что внутри |
|---|---|
| `main.go` | Роутер Gin, шаблоны, корректная остановка сервера |
| `handlers.go` | Страницы, `POST /api/lead`, анти-спам (honeypot + rate limit) |
| `leads.go` | Сохранение заявок в `data/leads.jsonl` и уведомления в Telegram |
| `content.go` | **Все тексты, цены, контакты и проекты портфолио.** Правьте здесь |
| `view.go` | SVG-иконки |
| `templates/` | `index.html` (студия), `404.html` |
| `static/` | CSS, JS, favicon |

## Страницы

- `/`: главная студии (услуги, обо мне, работы, цены, процесс, отзывы, FAQ, контакты)

Проекты в блоке «Работы» берутся из `Projects` в `content.go`. Если у проекта появится живая версия, укажите её адрес в поле `LiveURL`: на карточке появится кнопка «Открыть сайт».

## Заявки в Telegram

1. Создайте бота через @BotFather и получите токен.
2. Напишите боту что-нибудь, затем узнайте свой `chat_id` через `https://api.telegram.org/bot<TOKEN>/getUpdates`.
3. Запустите сервер с переменными окружения:

```bash
TELEGRAM_BOT_TOKEN=xxx TELEGRAM_CHAT_ID=yyy go run .
```

Без них заявки просто сохраняются в `data/leads.jsonl`.

## Продакшен

```bash
GIN_MODE=release PORT=8080 go build -o ar-web-studio . && ./ar-web-studio
```

Бинарнику нужны рядом папки `templates/` и `static/`.
