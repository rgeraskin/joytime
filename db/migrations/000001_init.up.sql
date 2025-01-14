-- Таблица семей
CREATE TABLE Families (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    uid VARCHAR(100) NOT NULL, -- Уникальный идентификатор
    created_by BIGINT, -- Временное поле (добавим внешний ключ позже)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_uid UNIQUE (uid) -- Уникальность значения
);

-- Таблица пользователей
CREATE TABLE Users (
    id SERIAL PRIMARY KEY,
    tg_id BIGINT NOT NULL,
    role VARCHAR(10) CHECK (role IN ('parent', 'child')) NOT NULL,
    family_uid VARCHAR(100) REFERENCES Families(uid) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    text_input_for VARCHAR(100),
    text_input_arg VARCHAR(100),
    CONSTRAINT unique_tg_id UNIQUE (tg_id) -- Уникальность значения
);

-- Добавление внешнего ключа на created_by после создания таблицы Users
-- ALTER TABLE Families
-- ADD CONSTRAINT fk_families_created_by FOREIGN KEY (created_by) REFERENCES Users(tg_id);

-- Таблица заданий
CREATE TABLE Tasks (
    id SERIAL PRIMARY KEY,
    family_uid VARCHAR(100) REFERENCES Families(uid) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    tokens INT NOT NULL CHECK (tokens > 0),
    status VARCHAR(20) CHECK (status IN ('new', 'check')) DEFAULT 'new',
    one_off BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_tasks_family_uid_name UNIQUE (family_uid, name) -- Уникальность значения
);

-- Таблица наград
CREATE TABLE Rewards (
    id SERIAL PRIMARY KEY,
    family_uid VARCHAR(100) REFERENCES Families(uid) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    tokens INT NOT NULL CHECK (tokens > 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_rewards_family_uid_name UNIQUE (family_uid, name) -- Уникальность значения
);

-- Таблица жетонов (учёт баланса пользователя)
CREATE TABLE Tokens (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES Users(tg_id) ON DELETE CASCADE,
    balance INT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для ускорения запросов
CREATE INDEX idx_users_family_uid ON Users(family_uid);
CREATE INDEX idx_tasks_family_uid ON Tasks(family_uid);
CREATE INDEX idx_rewards_family_uid ON Rewards(family_uid);