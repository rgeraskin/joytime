🇷🇺 **Русский** | 🇬🇧 [English](README.en.md)

# JoyTime

Семейное приложение для управления задачами и наградами с токен-экономикой. Родители назначают задания, дети выполняют их за токены, токены обмениваются на награды. Основной интерфейс — Telegram-бот, также доступен REST API.

> Бот вдохновлён идеями [Б. Ф. Скиннера](https://ru.wikipedia.org/wiki/%D0%A1%D0%BA%D0%B8%D0%BD%D0%BD%D0%B5%D1%80,_%D0%91%D0%B5%D1%80%D1%80%D0%B5%D1%81_%D0%A4%D1%80%D0%B5%D0%B4%D0%B5%D1%80%D0%B8%D0%BA) — системой воспитания через подкрепления и последствия — и современной теорией геймификации. Жетонная экономика, двухэтапное одобрение родителем и штрафы как response cost — прямые отсылки к поведенческому анализу. Подробнее — в разделе [Теоретическая основа](#теоретическая-основа).

## Основные функции

- Управление семьями и пользователями (родители/дети)
- Одноразовые коды приглашения с deep link (`t.me/bot?start=CODE`)
- Создание и управление заданиями, наградами и штрафами
- Система токенов с полной историей операций
- Задания повторяемые — сбрасываются после одобрения родителем
- Двухэтапная проверка заданий: ребёнок отмечает → родитель подтверждает/отклоняет
- Массовый импорт заданий/наград/штрафов списком
- Ручная коррекция токенов с указанием причины
- Уведомления между родителями и детьми
- Управление семьёй: переименование и удаление участников
- История токенов для родителя и ребёнка
- RBAC+ABAC авторизация через Casbin (роли + изоляция семей)
- SQLite по умолчанию, PostgreSQL опционально
- REST API (`/api/v1/`) для интеграции с web/mobile
- Telegram-бот с inline-клавиатурами и эмодзи

## Быстрый старт

### Требования

- [mise](https://mise.jdx.dev/) — автоматически установит Go 1.23+
- Docker (только для PostgreSQL, необязателен для SQLite)

### Установка и запуск

```bash
git clone <repository-url>
cd joytime

mise install              # Установить зависимости
mise run setup-env        # Создать .env файл (указать TOKEN бота)
mise run run              # Собрать и запустить (SQLite + Telegram бот + HTTP на :8080)
```

Для PostgreSQL:
```bash
mise run db:up            # Запустить PostgreSQL
# В .env: DB_TYPE=postgres + PG* переменные
mise run run
```

### Тестирование

```bash
mise run test             # Все тесты (автоматически поднимает PostgreSQL)
mise run test-coverage    # Отчет покрытия
mise run ci               # Формат + линт + тесты
```

Отдельный тест: `go test ./internal/handlers/ -run TestName -v` (БД должна быть запущена).

## Telegram-бот

### Регистрация

- `/start` → «Создать семью» или «Ввести код приглашения»
- Deep link: `t.me/botname?start=CODE` — автоматический вход по ссылке
- Код приглашения одноразовый, содержит роль (родитель/ребёнок) и семью

### Родитель

- Главное меню: баланс детей, кнопка проверки заданий
- **Задания**: список (по убыванию токенов), добавить (по одному или списком), изменить цену, удалить
- **Награды**: список, добавить (по одному или списком), изменить цену, удалить
- **Штрафы**: список, добавить (по одному или списком), изменить цену, удалить, применить к ребёнку
- **Коррекция**: ручное начисление/списание токенов с указанием причины
- **Семья**: пригласить (родитель/ребёнок), переименовать, удалить участника
- **История**: просмотр истории токенов по ребёнку (последние 20 записей)
- **Проверка**: одобрить или отклонить выполненные ребёнком задания

### Ребёнок

- Главное меню: баланс токенов
- **Выполнить задание**: выбрать задание → отправить на проверку родителю
- **Получить награду**: выбрать награду → списать токены
- **Штрафы**: просмотр списка (только чтение)
- **История**: просмотр своей истории токенов

### Массовый импорт

Задания, награды и штрафы можно добавлять списком — каждый на новой строке, последнее слово — количество токенов:

```
Загрузить посудомойку 2
Вынести мусор 5
Читать час 12
```

## API

Все эндпоинты требуют заголовок `X-User-ID` для аутентификации (кроме `/api/v1/health`).

### Семьи

```
GET    /api/v1/families          # Список семей
POST   /api/v1/families          # Создать семью (UID генерируется автоматически)
GET    /api/v1/families/{uid}    # Получить семью
PUT    /api/v1/families/{uid}    # Обновить семью
```

### Пользователи

```
GET    /api/v1/users             # Список пользователей семьи
GET    /api/v1/users/{userID}    # Получить пользователя
PUT    /api/v1/users/{userID}    # Обновить пользователя
DELETE /api/v1/users/{userID}    # Удалить пользователя
```

### Задания

```
POST   /api/v1/tasks                          # Создать задание
GET    /api/v1/tasks/{familyUID}              # Задания семьи
GET    /api/v1/tasks/{familyUID}/{taskName}   # Конкретное задание
PUT    /api/v1/tasks/{familyUID}/{taskName}   # Обновить задание
DELETE /api/v1/tasks/{familyUID}/{taskName}   # Удалить задание
POST   /api/v1/tasks/{familyUID}/{taskName}   # Завершить задание
```

### Награды

```
POST   /api/v1/rewards                            # Создать награду
GET    /api/v1/rewards/{familyUID}                # Награды семьи
GET    /api/v1/rewards/{familyUID}/{rewardName}   # Конкретная награда
PUT    /api/v1/rewards/{familyUID}/{rewardName}   # Обновить награду
DELETE /api/v1/rewards/{familyUID}/{rewardName}   # Удалить награду
```

### Токены

```
POST   /api/v1/tokens                  # Начислить/списать токены
GET    /api/v1/tokens/users/{userID}   # Баланс пользователя
POST   /api/v1/tokens/users/{userID}   # Обновить баланс
GET    /api/v1/token-history           # Вся история
GET    /api/v1/token-history/{userID}  # История пользователя
```

## Архитектура

Трёхуровневая архитектура в `internal/`:

```
cmd/
├── main.go                    # Точка входа (HTTP + Telegram)
└── config.go                  # Конфигурация (SQLite/PostgreSQL)

internal/
├── handlers/                  # HTTP-слой
│   ├── http.go                # Маршруты и сервер
│   ├── middleware.go           # Auth middleware (X-User-ID → AuthContext)
│   ├── handler_families.go    # Эндпоинты семей
│   ├── handler_users.go       # Эндпоинты пользователей
│   ├── handler_tasks.go       # Эндпоинты заданий
│   ├── handler_rewards.go     # Эндпоинты наград
│   ├── handler_tokens.go      # Эндпоинты токенов
│   ├── types.go               # Типы, хелперы ответов
│   └── validation.go          # Валидация
│
├── telegram/                  # Telegram-бот
│   ├── bot.go                 # Bot struct, маршрутизация, хелперы
│   ├── registration.go        # Регистрация, deep link, приглашения
│   ├── parent.go              # Меню родителя, CRUD задания/награды/штрафы
│   └── child.go               # Меню ребёнка, выполнение, получение наград
│
├── domain/                    # Бизнес-логика
│   ├── services.go            # Агрегатор всех сервисов
│   ├── service_task.go        # TaskService (CRUD + complete/reject)
│   ├── service_reward.go      # RewardService (CRUD)
│   ├── service_penalty.go     # PenaltyService (CRUD + apply)
│   ├── service_token.go       # TokenService (баланс, история, claim)
│   ├── service_user.go        # UserService (профили, AuthContext, состояние)
│   ├── service_family.go      # FamilyService (семьи, UID генерация)
│   ├── service_invite.go      # InviteService (одноразовые коды приглашения)
│   ├── casbin_auth.go         # Casbin RBAC+ABAC (программная модель)
│   ├── types.go               # AuthContext, DTO, ошибки, валидация
│   └── utils.go               # UpdateFields хелпер
│
├── database/                  # Подключение к БД
│   ├── client.go              # GORM setup (SQLite/PostgreSQL), миграция
│   └── fill.go                # Тестовые данные
│
└── models/                    # GORM модели
    └── schema.go              # Families, Users, Tasks, Rewards, Penalties, Invites, Tokens, TokenHistory
```

## Окружение

Настраивается через `.env` (см. `mise run setup-env`):

| Переменная   | По умолчанию  | Описание                                   |
|--------------|---------------|--------------------------------------------|
| `TOKEN`      | (обязательно) | Токен Telegram-бота                        |
| `DB_TYPE`    | `sqlite`      | Тип БД: `sqlite` или `postgres`            |
| `DB_PATH`    | `joytime.db`  | Путь к файлу SQLite                        |
| `PGHOST`     | —             | Хост PostgreSQL (обязательно при postgres) |
| `PGPORT`     | —             | Порт PostgreSQL                            |
| `PGUSER`     | —             | Пользователь PostgreSQL                    |
| `PGPASSWORD` | —             | Пароль PostgreSQL                          |
| `PGDATABASE` | —             | Имя базы PostgreSQL                        |
| `DEBUG`      | —             | Включает отладочное логирование            |

## Полезные команды

```bash
mise run build              # Сборка
mise run run                # Сборка и запуск
mise run test               # Тесты
mise run fmt                # Форматирование
mise run lint               # Линтер
mise run db:up              # Запустить PostgreSQL
mise run db:down            # Остановить PostgreSQL
mise run db:reset           # Полный сброс с тестовыми данными
mise run db:dump [file]     # Дамп базы (по умолчанию dump.sql)
mise run db:restore [file]  # Восстановление базы
mise run db:shell           # psql в базу данных
```

## Теоретическая основа

JoyTime — практическое воплощение двух связанных традиций: **жетонной экономики** из бихевиоризма Б. Ф. Скиннера и **геймификации** из современной теории мотивации.

### Почему это работает

Если отвлечься от терминологии, подход держится на пяти простых механизмах:

1. **Поведение становится видимым.** «Убрать посуду» — это абстракция. «Получить 2 токена» — конкретный, отслеживаемый сигнал. Прогресс в цифрах понятен и ребёнку, и родителю.
2. **Награда приходит сразу.** Бот реагирует в течение секунд после одобрения. Мозг связывает действие и результат, пока память о нём свежая. Фразы вроде «потом скажешь мне спасибо» так не работают — мозг так не устроен.
3. **Токены учат терпению.** Ребёнок может копить на большую награду вместо мелких трат — это тренировка того же навыка, что измерялся в знаменитом «маршмеллоу-тесте»: умения подождать ради бóльшего.
4. **Конфликт становится безличным.** Вместо «ты опять не сделал!» — «ты сегодня не заработал». Правила зафиксированы в системе, не зависят от настроения родителя и не превращаются в ежедневные пререкания. Это снижает эмоциональный градус с обеих сторон.
5. **У ребёнка появляется выбор.** Он сам решает, какие задания брать и на что тратить токены. Даже небольшой выбор меняет отношение с «меня заставляют» на «я зарабатываю».

### Связь со Скиннером

JoyTime — по сути **цифровая жетонная система**, прямой потомок техники Ayllon & Azrin (1968), которая изначально применялась в психиатрических отделениях и коррекционных классах. Архитектура почти один-в-один ложится на трёхчленную контингентность Скиннера: задача — дискриминативный стимул, выполнение — оперантное поведение, начисление/списание токенов — подкрепление/наказание.

- **Жетонная экономика (token economy)** — методика, сформулированная Ayllon & Azrin (1968) на основе оперантного обусловливания Скиннера: нейтральные токены становятся условными подкрепителями через обмен на ценные вещи. В коде это реализовано парой `TokenService` + `RewardService` — зарабатывание отделено от траты.
- **Трёхчленная контингентность** (антецедент → поведение → последствие; Skinner, 1953): назначенная задача — дискриминативный стимул, выполнение — оперантное поведение, начисление токена — подкрепитель.
- **Response cost (отрицательное наказание)**: `PenaltyService` изымает токены — поведенчески эффективная и более гуманная альтернатива положительному наказанию (Kazdin, 1972; Pazulinec, Meyerrose & Sajwaj, 1983).
- **Шейпинг**: цена задач в токенах градуируется по сложности — принцип последовательных приближений.
- **Аудит-лог как кумулятивный регистратор**: `TokenHistory` — цифровой аналог прибора, изобретённого Скиннером для анализа частоты реакций.

### Связь с геймификацией

По классификации Werbach & Hunter (2012) «PBL» (Points / Badges / Leaderboards) JoyTime реализует **Points**, сознательно избегая **Leaderboards**: в семейном контексте рейтинг между сиблингами превращает кооперацию в конкуренцию и демотивирует слабого участника (Hamari, Koivisto & Sarsa, 2014; Landers et al., 2017).

По теории самодетерминации (Deci & Ryan, 1985) устойчивая мотивация требует трёх компонентов:

| Компонент                   | Статус в JoyTime                                |
|-----------------------------|-------------------------------------------------|
| Автономия (autonomy)        | Слабо — задачи назначаются родителем            |
| Компетентность (competence) | Слабо — нет уровней и видимой прогрессии        |
| Связанность (relatedness)   | Сильно — семейный контекст, одобрение родителем |

Из 8 Core Drives фреймворка Octalysis (Chou, 2015) активны #2 Development & Accomplishment, #4 Ownership & Possession, #5 Social Influence и #8 Loss & Avoidance; #7 Unpredictability пока отсутствует.

### Известные риски

- **Overjustification effect** (Deci, 1971; Lepper, Greene & Nisbett, 1973): сильные внешние вознаграждения могут снижать внутреннюю мотивацию к занятиям, которые ребёнку уже нравились.
- **Gaming the system**: дети оптимизируют под метрику, а не под цель — частично нейтрализуется двухэтапным одобрением родителем.
- **Pointsification** (Robertson, 2010): простое навешивание очков без нарратива и прогрессии не даёт долгосрочного вовлечения — открытая зона роста для продукта.

### Литература

- Ayllon, T., & Azrin, N. H. (1968). *The Token Economy: A Motivational System for Therapy and Rehabilitation.* Appleton-Century-Crofts.
- Skinner, B. F. (1953). *Science and Human Behavior.* Macmillan.
- Kazdin, A. E. (1972). Response cost: The removal of conditioned reinforcers for therapeutic change. *Behavior Therapy*, 3(4), 533–546. doi:10.1016/S0005-7894(72)80003-8
- Deci, E. L. (1971). Effects of externally mediated rewards on intrinsic motivation. *Journal of Personality and Social Psychology*, 18(1), 105–115. doi:10.1037/h0030644
- Lepper, M. R., Greene, D., & Nisbett, R. E. (1973). Undermining children's intrinsic interest with extrinsic reward. *JPSP*, 28(1), 129–137. doi:10.1037/h0035519
- Deci, E. L., & Ryan, R. M. (1985). *Intrinsic Motivation and Self-Determination in Human Behavior.* Plenum.
- Pazulinec, R., Meyerrose, M., & Sajwaj, T. (1983). Punishment via response cost. In S. Axelrod & J. Apsche (Eds.), *The Effects of Punishment on Human Behavior* (pp. 71–86). Academic Press.
- Csikszentmihalyi, M. (1990). *Flow: The Psychology of Optimal Experience.* Harper & Row.
- Deterding, S., Dixon, D., Khaled, R., & Nacke, L. (2011). From game design elements to gamefulness: defining "gamification". *MindTrek '11.* doi:10.1145/2181037.2181040
- Werbach, K., & Hunter, D. (2012). *For the Win: How Game Thinking Can Revolutionize Your Business.* Wharton Digital Press.
- Hamari, J., Koivisto, J., & Sarsa, H. (2014). Does gamification work? — A literature review of empirical studies on gamification. *HICSS 2014.* doi:10.1109/HICSS.2014.377
- Chou, Y.-k. (2015). *Actionable Gamification: Beyond Points, Badges, and Leaderboards.* Octalysis Media.
- Landers, R. N., Bauer, K. N., & Callan, R. C. (2017). Gamification of task performance with leaderboards: A goal setting experiment. *Computers in Human Behavior*, 71, 508–515. doi:10.1016/j.chb.2015.08.008
- Kohn, A. (1993). *Punished by Rewards: The Trouble with Gold Stars, Incentive Plans, A's, Praise, and Other Bribes.* Houghton Mifflin.

## TODO

1. i18n
1. Telegram miniApp
1. Variable-ratio bonus rewards (random surprise token bonus on task approval — Skinner VR schedule)
