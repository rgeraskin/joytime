🇷🇺 [Русский](README.md) | 🇬🇧 **English**

# JoyTime

> [!NOTE]
> The Telegram bot's user interface is currently available in Russian only. English localization of the bot and API response messages is on the roadmap but not yet implemented.

A family app for managing tasks and rewards with a token economy. Parents assign tasks, children complete them for tokens, and tokens are exchanged for rewards. The primary interface is a Telegram bot; a REST API is also available.

> This bot is inspired by the ideas of [B. F. Skinner](https://en.wikipedia.org/wiki/B._F._Skinner) — a parenting approach based on reinforcement and consequences — and modern gamification theory. The token economy, two-stage parent approval, and penalties as response cost are direct references to behavior analysis. More in the [Theoretical Foundation](#theoretical-foundation) section.

## Features

- Family and user management (parents/children)
- One-time invite codes with deep links (`t.me/bot?start=CODE`)
- Create and manage tasks, rewards, and penalties
- Token system with full operation history
- Repeatable tasks — reset after parent approval
- Two-stage task verification: child marks → parent approves/rejects
- Bulk import of tasks/rewards/penalties as a list
- Manual token adjustment with reason
- Notifications between parents and children
- Family management: rename and remove members
- Token history for parent and child
- RBAC+ABAC authorization via Casbin (roles + family isolation)
- SQLite by default, PostgreSQL optional
- REST API (`/api/v1/`) for web/mobile integration
- Telegram bot with inline keyboards and emojis

## Quick Start

### Requirements

- [mise](https://mise.jdx.dev/) — automatically installs Go 1.23+
- Docker (only for PostgreSQL, not required for SQLite)

### Installation and Running

```bash
git clone <repository-url>
cd joytime

mise install              # Install dependencies
mise run setup-env        # Create .env file (set bot TOKEN)
mise run run              # Build and run (SQLite + Telegram bot + HTTP on :8080)
```

For PostgreSQL:
```bash
mise run db:up            # Start PostgreSQL
# In .env: DB_TYPE=postgres + PG* variables
mise run run
```

### Testing

```bash
mise run test             # All tests (automatically starts PostgreSQL)
mise run test-coverage    # Coverage report
mise run ci               # Format + lint + tests
```

Single test: `go test ./internal/handlers/ -run TestName -v` (DB must be running).

## Telegram Bot

### Registration

- `/start` → "Create family" or "Enter invite code"
- Deep link: `t.me/botname?start=CODE` — automatic login via link
- Invite code is one-time, contains role (parent/child) and family

### Parent

- Main menu: child balances, task review button
- **Tasks**: list (sorted by tokens descending), add (one by one or as list), change price, delete
- **Rewards**: list, add (one by one or as list), change price, delete
- **Penalties**: list, add (one by one or as list), change price, delete, apply to child
- **Adjust**: manual token credit/debit with reason
- **Family**: invite (parent/child), rename, remove member
- **History**: view token history per child (last 20 entries)
- **Review**: approve or reject tasks completed by the child

### Child

- Main menu: token balance
- **Complete task**: select task → send for parent review
- **Claim reward**: select reward → deduct tokens
- **Penalties**: view list (read-only)
- **History**: view own token history

### Bulk Import

Tasks, rewards, and penalties can be added as a list — one per line, last word is the token amount:

```
Load dishwasher 2
Take out trash 5
Read for an hour 12
```

## API

All endpoints require the `X-User-ID` header for authentication (except `/api/v1/health`).

### Families

```
GET    /api/v1/families          # List families
POST   /api/v1/families          # Create family (UID generated automatically)
GET    /api/v1/families/{uid}    # Get family
PUT    /api/v1/families/{uid}    # Update family
```

### Users

```
GET    /api/v1/users             # List family users
GET    /api/v1/users/{userID}    # Get user
PUT    /api/v1/users/{userID}    # Update user
DELETE /api/v1/users/{userID}    # Delete user
```

### Tasks

```
POST   /api/v1/tasks                          # Create task
GET    /api/v1/tasks/{familyUID}              # Family's tasks
GET    /api/v1/tasks/{familyUID}/{taskName}   # Specific task
PUT    /api/v1/tasks/{familyUID}/{taskName}   # Update task
DELETE /api/v1/tasks/{familyUID}/{taskName}   # Delete task
POST   /api/v1/tasks/{familyUID}/{taskName}   # Complete task
```

### Rewards

```
POST   /api/v1/rewards                            # Create reward
GET    /api/v1/rewards/{familyUID}                # Family's rewards
GET    /api/v1/rewards/{familyUID}/{rewardName}   # Specific reward
PUT    /api/v1/rewards/{familyUID}/{rewardName}   # Update reward
DELETE /api/v1/rewards/{familyUID}/{rewardName}   # Delete reward
```

### Tokens

```
POST   /api/v1/tokens                  # Credit/debit tokens
GET    /api/v1/tokens/users/{userID}   # User balance
POST   /api/v1/tokens/users/{userID}   # Update balance
GET    /api/v1/token-history           # Full history
GET    /api/v1/token-history/{userID}  # User history
```

## Architecture

Three-layer architecture under `internal/`:

```
cmd/
├── main.go                    # Entry point (HTTP + Telegram)
└── config.go                  # Configuration (SQLite/PostgreSQL)

internal/
├── handlers/                  # HTTP layer
│   ├── http.go                # Routes and server
│   ├── middleware.go          # Auth middleware (X-User-ID → AuthContext)
│   ├── handler_families.go    # Family endpoints
│   ├── handler_users.go       # User endpoints
│   ├── handler_tasks.go       # Task endpoints
│   ├── handler_rewards.go     # Reward endpoints
│   ├── handler_tokens.go      # Token endpoints
│   ├── types.go               # Types, response helpers
│   └── validation.go          # Validation
│
├── telegram/                  # Telegram bot
│   ├── bot.go                 # Bot struct, routing, helpers
│   ├── registration.go        # Registration, deep link, invites
│   ├── parent.go              # Parent menu, CRUD tasks/rewards/penalties
│   └── child.go               # Child menu, task completion, reward claim
│
├── domain/                    # Business logic
│   ├── services.go            # Services aggregator
│   ├── service_task.go        # TaskService (CRUD + complete/reject)
│   ├── service_reward.go      # RewardService (CRUD)
│   ├── service_penalty.go     # PenaltyService (CRUD + apply)
│   ├── service_token.go       # TokenService (balance, history, claim)
│   ├── service_user.go        # UserService (profiles, AuthContext, state)
│   ├── service_family.go      # FamilyService (families, UID generation)
│   ├── service_invite.go      # InviteService (one-time invite codes)
│   ├── casbin_auth.go         # Casbin RBAC+ABAC (programmatic model)
│   ├── types.go               # AuthContext, DTOs, errors, validation
│   └── utils.go               # UpdateFields helper
│
├── database/                  # DB connection
│   ├── client.go              # GORM setup (SQLite/PostgreSQL), migration
│   └── fill.go                # Test data
│
└── models/                    # GORM models
    └── schema.go              # Families, Users, Tasks, Rewards, Penalties, Invites, Tokens, TokenHistory
```

## Environment

Configured via `.env` (see `mise run setup-env`):

| Variable     | Default       | Description                                |
|--------------|---------------|--------------------------------------------|
| `TOKEN`      | (required)    | Telegram bot token                         |
| `DB_TYPE`    | `sqlite`      | DB type: `sqlite` or `postgres`            |
| `DB_PATH`    | `joytime.db`  | Path to SQLite file                        |
| `PGHOST`     | —             | PostgreSQL host (required when postgres)   |
| `PGPORT`     | —             | PostgreSQL port                            |
| `PGUSER`     | —             | PostgreSQL user                            |
| `PGPASSWORD` | —             | PostgreSQL password                        |
| `PGDATABASE` | —             | PostgreSQL database name                   |
| `DEBUG`      | —             | Enables debug logging                      |

## Useful Commands

```bash
mise run build              # Build
mise run run                # Build and run
mise run test               # Run tests
mise run fmt                # Format code
mise run lint               # Lint
mise run db:up              # Start PostgreSQL
mise run db:down            # Stop PostgreSQL
mise run db:reset           # Full reset with test data
mise run db:dump [file]     # Dump database (default dump.sql)
mise run db:restore [file]  # Restore database
mise run db:shell           # psql into the database
```

## Theoretical Foundation

JoyTime is a practical embodiment of two connected traditions: the **token economy** from B. F. Skinner's behaviorism and **gamification** from modern motivation theory.

### Why It Works

Setting terminology aside, the approach rests on five simple mechanisms:

1. **Behavior becomes visible.** "Put away the dishes" is an abstraction. "Earn 2 tokens" is a concrete, trackable signal. Progress in numbers is legible to both child and parent.
2. **The reward arrives immediately.** The bot responds within seconds of approval. The brain connects action to result while the memory is still fresh. Phrases like "you'll thank me later" don't work — the brain isn't wired that way.
3. **Tokens teach patience.** A child can save up for a bigger reward instead of spending on small things — training the same skill measured in the famous "marshmallow test": the ability to wait for something greater.
4. **Conflict becomes impersonal.** Instead of "you didn't do it again!" it's "you didn't earn anything today." Rules are fixed in the system, don't depend on parental mood, and don't turn into daily bickering. This lowers the emotional temperature on both sides.
5. **The child gets a choice.** They decide which tasks to take on and what to spend tokens on. Even a small choice shifts the attitude from "I'm being forced" to "I'm earning."

### Alignment with Skinner

JoyTime is essentially a **digital token economy**, a direct descendant of the Ayllon & Azrin (1968) technique originally used in psychiatric wards and special-education classrooms. The architecture maps almost one-to-one onto Skinner's three-term contingency: task — discriminative stimulus, completion — operant behavior, token credit/debit — reinforcement/punishment.

- **Token economy** — a methodology formulated by Ayllon & Azrin (1968) based on Skinner's operant conditioning: neutral tokens become conditioned reinforcers through exchange for valuable items. In code this is realized by the `TokenService` + `RewardService` pair — earning is decoupled from spending.
- **Three-term contingency** (antecedent → behavior → consequence; Skinner, 1953): the assigned task is the discriminative stimulus, completion is the operant behavior, the token credit is the reinforcer.
- **Response cost (negative punishment)**: `PenaltyService` removes tokens — a behaviorally effective and more humane alternative to positive punishment (Kazdin, 1972; Pazulinec, Meyerrose & Sajwaj, 1983).
- **Shaping**: task prices in tokens are graduated by difficulty — the principle of successive approximations.
- **Audit log as cumulative recorder**: `TokenHistory` is the digital analog of the instrument Skinner invented to analyze response rates.

### Alignment with Gamification

Per the Werbach & Hunter (2012) "PBL" (Points / Badges / Leaderboards) classification, JoyTime implements **Points** while deliberately avoiding **Leaderboards**: in a family context, ranking between siblings turns cooperation into competition and demotivates the weaker participant (Hamari, Koivisto & Sarsa, 2014; Landers et al., 2017).

Per self-determination theory (Deci & Ryan, 1985), sustainable motivation requires three components:

| Component   | Status in JoyTime                                |
|-------------|--------------------------------------------------|
| Autonomy    | Weak — tasks are assigned by the parent          |
| Competence  | Weak — no levels or visible progression          |
| Relatedness | Strong — family context, parental approval       |

Of the 8 Core Drives in the Octalysis framework (Chou, 2015), the active ones are #2 Development & Accomplishment, #4 Ownership & Possession, #5 Social Influence, and #8 Loss & Avoidance; #7 Unpredictability is currently absent.

### Known Risks

- **Overjustification effect** (Deci, 1971; Lepper, Greene & Nisbett, 1973): strong external rewards can reduce intrinsic motivation for activities the child already enjoyed.
- **Gaming the system**: children optimize for the metric rather than the goal — partially neutralized by two-stage parent approval.
- **Pointsification** (Robertson, 2010): merely attaching points to things without narrative and progression doesn't produce long-term engagement — an open area for product growth.

### References

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
