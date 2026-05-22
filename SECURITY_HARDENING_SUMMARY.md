# Security hardening — кратко для рассказа

Что сделано: **без новых продуктовых фич**, только усиление безопасности и приватности API.

---

## Одна фраза для начала

«Добавили защитные HTTP-заголовки, убрали опору на IP из заголовков, подчистили логи, переделали rate limit без привязки к IP, в production — редирект на HTTPS и HSTS, опционально заголовок для Tor/onion; всё покрыто тестами.»

---

## По пунктам: что именно

### 1. Заголовки безопасности (каждый ответ)

- **CSP** — по сути «скрипты и ресурсы только со своего origin» (`default-src 'self'`).
- **X-Content-Type-Options: nosniff** — браузер не угадывает тип файла «на глаз».
- **X-Frame-Options: DENY** — страницу нельзя встроить во фрейм (кликджекинг сложнее).
- **Referrer-Policy: no-referrer** — не тащим полный URL в referrer на внешние сайты.
- **Permissions-Policy** — отключаем гео / микрофон / камеру для этого origin.

**Файл:** `internal/middleware/security.go`

---

### 2. HTTPS в production

- Если **`APP_ENV=production`** и прокси передал, что клиент **не по HTTPS** (`X-Forwarded-Proto` не `https`) → ответ **301** на тот же путь с `https://`.
- Если уже «как по HTTPS» → отдаётся **HSTS** (долго помнить только HTTPS).

**Важно для деплоя:** reverse proxy должен выставлять **`X-Forwarded-Proto`** так, как клиент реально пришёл снаружи.

**Файл:** `internal/middleware/security.go`

---

### 3. No-IP: вычищаем «IP из заголовков»

- Самым первым middleware **удаляем** заголовки: `X-Forwarded-For`, `X-Real-IP`, `CF-Connecting-IP`, `True-Client-IP`.
- В контекст кладётся пустой **`client_ip`**.
- В коде **нет вызовов `ClientIP()`**; есть тест, который сканирует репозиторий на это.
- В **development** после очистки есть проверка: если эти заголовки всё ещё есть → **panic** (ловим ошибки порядка middleware).

**Файлы:** `internal/middleware/noip.go`, `internal/audit/ip_audit.go`

---

### 4. Tor / onion (опционально)

- **Пустой `Server`**, убираем **`X-Powered-By`** — меньше «отпечатка» сервера.
- Если в `.env` задан **`ONION_ADDRESS`** — в ответ добавляется **`Onion-Location`** (подсказка Tor Browser перейти на `.onion`).

**Файл:** `internal/middleware/tor.go`

---

### 5. Rate limit без IP

- Раньше без JWT лимит мог считаться **по IP** (`ClientIP`).
- Теперь без токена все анонимы в одном ключе **`subject:anon`**; с JWT — по **хешу токена**.
- **Общий** лимит вешается **один раз** на весь роутер; отдельный жёсткий лимит только на **создание поста**.

**Файл:** `internal/middleware/ratelimit.go`  
**Порядок цепочки:** `cmd/server/main.go`

---

### 6. Логи HTTP

- Пишем только: **method, path, status, latency**, и **pseudonym** (если после auth он есть в контексте).
- **Не** логируем: IP, User-Agent, сырые заголовки.

**Файл:** `internal/middleware/logger.go` (вместо старого `requestlog.go`)

---

### 7. Конфиг

- **`APP_ENV`** — `development` (по умолчанию) или `production`.
- **`ONION_ADDRESS`** — опционально, для `Onion-Location`.

**Файлы:** `internal/config/config.go`, `.env.example`

---

### 8. Тесты

- **`internal/middleware/security_test.go`** — CSP, снятие XFF, 301 в production, Server не `gin`, лог без referer/IP.
- **`internal/audit/ip_audit_test.go`** — запрет на `.ClientIP(` в исходниках.

Команда: `go test ./...`

---

## Порядок middleware (для вопроса «почему так»)

1. NoIP — сначала убрать чувствительные заголовки.  
2. Recovery  
3. Security — заголовки + HTTPS/HSTS в prod  
4. Tor — Server / X-Powered-By / Onion-Location  
5. Rate limit (общий)  
6. Logger  
7. CORS  

---

## Что показать в Postman / curl

1. `GET /health` → в **ответе** видны CSP, X-Frame-Options, Referrer-Policy и т.д.  
2. В запрос добавить `X-Forwarded-For: 203.0.113.1` → ответ нормальный, в логах этого IP нет.  
3. (Демо) `APP_ENV=production`, в запросе `X-Forwarded-Proto: http` → **301** на `https://...`  
4. (Опционально) задать `ONION_ADDRESS` → в ответе есть **Onion-Location**.

---

## Зачем проекту (если спросят «зачем»)

| Тема            | Польза                                      |
|-----------------|---------------------------------------------|
| Заголовки       | Меньше XSS, кликджекинга, лишних утечек URL |
| No-IP + логи    | Меньше хранения/утечки реальных IP          |
| Rate limit      | Защита API без привязки к IP                |
| HTTPS + HSTS    | Сильнее транспорт в production              |
| Tor / onion     | Дружелюбнее к Tor, опция для .onion        |
| Тесты           | Регрессии сложнее занести                   |

---

## Список новых/важных файлов (для PR)

- `internal/middleware/noip.go`
- `internal/middleware/security.go`
- `internal/middleware/tor.go`
- `internal/middleware/logger.go`
- `internal/audit/ip_audit.go`
- `internal/middleware/security_test.go`
- `internal/audit/ip_audit_test.go`

Изменены: `internal/config/config.go`, `internal/middleware/ratelimit.go`, `cmd/server/main.go`, `.env.example`  
Удалён: `internal/middleware/requestlog.go`

---

*Файл для личной подготовки к рассказу; в репозиторий можно не коммитить, если не нужен команде — тогда добавь в `.gitignore` или удали после использования.*
