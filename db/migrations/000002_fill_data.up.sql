INSERT INTO public.families (uid,created_by) VALUES ('sa726q', 220328701);
INSERT INTO public.users (tg_id,role,family_uid) VALUES (220328701, 'parent', 'sa726q');

INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Загрузить посудомойку', 2);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Достать и расставить посуду из посудомойки', 2);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Помыть посуду у папы', 5);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Постирать', 2);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Вынести мусор', 5);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Покормить кошку', 2);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Почистить туалет кошки', 2);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Занятие по шахматам', 10);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ('sa726q', 'Занятие по шахматам онлайн', 5);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ( 'sa726q', 'Занятие по футболу', 10);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ( 'sa726q', 'Турнир по шахматам', 30);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ( 'sa726q', 'Читать час', 12);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ( 'sa726q', 'Побрызгаться дезодорантом', 1);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ( 'sa726q', 'Поставить будильник и самому по нему проснуться', 1);
INSERT INTO public.tasks (family_uid,name,tokens) VALUES ( 'sa726q', 'Приготовить себе еду', 6);

INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Смотреть YouTube/VK 15м', 5);
INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Смотреть YouTube/VK 60м', 20);
INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Играть в Роблокс/Melon 15м', 4);
INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Играть в Роблокс/Melon 60м', 16);
INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 15м', 2);
INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Играть в Шахматы / Human Resource Machine / Cargo-Bot / Hearthstone 60м', 8);
INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Играть в остальные игры 15м', 3);
INSERT INTO public.rewards (family_uid,name,tokens) VALUES ('sa726q', 'Играть в остальные игры 60м', 12);