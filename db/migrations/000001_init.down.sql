-- Удалить внешние ключи, которые ссылаются на Families
ALTER TABLE Users DROP CONSTRAINT IF EXISTS users_family_id_fkey;
ALTER TABLE Families DROP CONSTRAINT IF EXISTS fk_families_created_by;
ALTER TABLE Tokens DROP CONSTRAINT IF EXISTS tokens_user_id_fkey;

-- Удалить таблицы в обратном порядке создания
DROP TABLE IF EXISTS Tokens;
DROP TABLE IF EXISTS Rewards;
DROP TABLE IF EXISTS Tasks;
DROP TABLE IF EXISTS Users;
DROP TABLE IF EXISTS Families;