# BotFather setup

This guide covers configuring @BotFather for the **Lead Cat** Google Meet meetings bot.
For environment variables and deployment, see `docs/SETUP.md` and `docs/DEPLOY-DOKPLOY.md`.

---

## 1. Create the bot

1. Open [@BotFather](https://t.me/BotFather) and send `/newbot`.
2. Enter a display name, e.g. **Lead Cat**.
3. Enter a username ending in `bot`, e.g. `leadcat_bot`.
4. Copy the token BotFather returns and store it as `BOT_TOKEN` in your environment.

> **Never commit a real token.** Use a placeholder or a secret manager.

---

## 2. Set the Mini App (Web App) URL

The bot's main menu button opens the Mini App directly in Telegram.

1. In @BotFather select your bot → **Menu Button** → **Edit menu button**.
2. Choose **Web App** and enter the deployed Mini App URL.
   Store this URL as `WEBAPP_URL` in your environment (e.g. `https://meetings.example.com`).
3. Alternatively, use `/newapp` to register a named Web App entry point in @BotFather.

When a user taps the bot's menu button, Telegram opens `WEBAPP_URL` as a Telegram Mini App.

---

## 3. Set bot commands (`/setcommands`)

Run `/setcommands` in @BotFather, select your bot, then paste the list below.
Commands are in Russian to match the product spec (ТЗ §8).

```
start - Запуск бота; регистрация или главное меню
menu - Открыть главное меню
new - Создать новую встречу
my_meetings - Просмотр своих встреч
schedule - Просмотр расписания сотрудника
checker - Чекер общего свободного времени
settings - Открыть настройки пользователя
help - Справка по командам бота
admin - Панель администратора (только для администраторов)
```

> `/admin` is only functional for users with the admin role; all other users receive a
> permission-denied response.

---

## 4. `/start` — registration

When a user sends `/start` the backend upserts a `bot_user` row that binds their
`telegram_id` to an email account. If no account exists yet, the bot prompts the user to
register or link an existing account via the Mini App.

See `docs/AUTH.md` for the full authentication and identity-linking flow.

---

## 5. Test bot vs production bot (optional)

Use a **separate BotFather token** for local development and staging:

| Environment | `BOT_TOKEN`            | `WEBAPP_URL`                     |
| ----------- | ---------------------- | -------------------------------- |
| Production  | `<prod token>`         | `https://meetings.example.com`   |
| Dev / CI    | `<dev token>`          | `https://localhost:5173` (or ngrok URL) |

This prevents test traffic from hitting production users and allows webhook re-registration
without downtime.
