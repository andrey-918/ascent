# AI Planner

Учебный backend-проект приложения для целей и дневниковых записей. Идея продукта — позволить пользователю описывать события обычным текстом, а позднее автоматически извлекать из него данные для прогресса по целям.

## Текущее состояние

Сейчас реализованы Go REST API, регистрация и вход с JWT, PostgreSQL, дневниковые записи и базовые финансовые цели. Защищённые маршруты требуют access token; пользователь может работать только со своими записями и целями.

### Реализованные маршруты

| Метод | Путь | Назначение | Успешный ответ |
|---|---|---|---|
| `POST` | `/auth/signup` | Регистрация с `email` и `password` | `201 Created` |
| `POST` | `/auth/signin` | Вход; выдаёт access и refresh tokens | `200 OK` |
| `POST` | `/auth/refresh` | Обновление пары токенов | `200 OK` |
| `POST` | `/diary-entries` | Создать дневниковую запись | `201 Created` |
| `GET` | `/diary-entries` | Получить страницу своих записей | `200 OK` |
| `GET` | `/diary-entries/{id}` | Получить свою запись | `200 OK` |
| `DELETE` | `/diary-entries/{id}` | Удалить свою запись | `204 No Content` |
| `POST` | `/goals` | Создать финансовую цель | `201 Created` |
| `GET` | `/goals` | Получить свои цели | `200 OK` |
| `GET` | `/goals/{id}` | Получить свою цель | `200 OK` |
| `GET` | `/ping` | Проверка доступности API | `200 OK` |

Ошибки прикладных обработчиков возвращаются JSON-ответом. Неверные параметры и UUID дают `400 Bad Request`; отсутствующая запись или цель, включая чужой объект, даёт `404 Not Found`; запрос без корректной авторизации — `401 Unauthorized`.

Список дневника упорядочен от новых записей к старым. По умолчанию используется `limit=20` и `offset=0`; допустимы `limit` от 1 до 100 и неотрицательный `offset`. Длина содержимого записи ограничена 10 000 символами.

Первая версия целей поддерживает только RUB. Суммы передаются целым количеством копеек, например 5 000 000 копеек — это 50 000 рублей. Новая цель создаётся с нулевым прогрессом и статусом `active`.

## Ручная проверка API

Ниже приведены примеры для `curl`. Для входа и создания записей сначала запусти API и PostgreSQL (инструкция ниже). Примеры используют `jq` для чтения токена и ID из JSON-ответов.

Задай адрес API:

```sh
export API_URL=http://localhost:8080
```

Создай пользователя:

```sh
curl -i -X POST "$API_URL/auth/signup" \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"change-this-password"}'
```

Войди и сохрани access token:

```sh
export ACCESS_TOKEN="$(
  curl -sS -X POST "$API_URL/auth/signin" \
    -H 'Content-Type: application/json' \
    -d '{"email":"demo@example.com","password":"change-this-password"}' \
  | jq -r '.data.access_token'
)"
```

Создай запись и сохрани её ID:

```sh
export ENTRY_ID="$(
  curl -sS -X POST "$API_URL/diary-entries" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{"content":"Сегодня отложил 100 рублей"}' \
  | jq -r '.data.id'
)"
```

Получить первую страницу записей, следующую страницу и одну запись:

```sh
curl -i "$API_URL/diary-entries" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl -i "$API_URL/diary-entries?limit=10&offset=10" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl -i "$API_URL/diary-entries/$ENTRY_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Создать цель накопить 50 000 рублей, получить список целей и одну цель:

```sh
export GOAL_ID="$(
  curl -sS -X POST "$API_URL/goals" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{"name":"Накопить 50 000 рублей","target_amount_kopecks":5000000}' \
  | jq -r '.data.id'
)"

curl -i "$API_URL/goals" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl -i "$API_URL/goals/$GOAL_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Удалить запись. При успехе сервер возвращает `204` без тела:

```sh
curl -i -X DELETE "$API_URL/diary-entries/$ENTRY_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Локальный запуск и тесты

Нужны Go версии, указанной в `go.mod`, Docker Compose и установленный `migrate` CLI для применения SQL-миграций. Настрой локальный `.env` с параметрами базы, API и JWT; не публикуй секреты и не используй development credentials вне локальной разработки.

```sh
docker compose up -d db
make db_migrate
go run ./cmd/server
```

В другом терминале можно проверить API через `curl` по примерам выше. Для тестов также нужна доступная PostgreSQL из локальной конфигурации:

```sh
go test ./...
```

## Планируемое развитие

Функции ниже пока не реализованы:

- события пополнения/снятия и пересчёт прогресса целей;
- пересчёт прогресса целей;
- Python worker и AI-анализ;
- асинхронная обработка через Kafka, ретраи и идемпотентность;
- метрики Prometheus и dashboards Grafana;
- контейнеризация всех компонентов и развёртывание в локальном Kubernetes.

Подробный план реализации находится в [todo.md](./todo.md).
