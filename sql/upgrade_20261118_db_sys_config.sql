-- 参数表 db_sys_config 加唯一约束 db_sys_config_sort_key_key：分组 sort 加键 key 不能重复（加完和 0001_create_tables.sql 里的一样）。
-- 程序按"分组.键"取参数的值，两行重复时起作用的只是 config_id 大的那一行；加上之后，新增重复的参数回"这个分组里已经有这个键"。
-- (sort, key) 上原来那个普通索引 idx_db_sys_config__sort_key 用不着了（唯一约束自带同样的索引），一起删掉。
-- 给参数表还没有这个约束的库执行一次。拿不准就执行：已经有了会停在开头的检查上，什么都不改。新建的库直接用 0001、0002，不用执行它。
-- 和别的升级脚本没有先后关系。
--
-- 执行之前：
--   1. 不用停服务，库和程序谁先换都行：程序先换的话，约束加上之前，重复的参数照样加得进去
--   2. 用服务连库的那个账号执行（要改表结构，得是表的所有者）
--   3. 表里已经有重复的参数时，脚本停在开头的检查上、什么都不改，并列出是哪几组：先删掉多余的行，再执行
--   4. 等锁最多 5 秒（有别的会话开着事务、碰过这张表时会等）：等不到就报错、什么都不改，过一会儿再执行。
--      不设上限的话，它排队等锁的这段时间，服务里读这张表的请求（取系统参数等）也都排在它后面
--
-- 做的事在一个事务里，中途出错什么都不改。
SET client_encoding TO 'UTF8';

BEGIN;
SET LOCAL lock_timeout = '5s';
LOCK TABLE db_sys_config IN ACCESS EXCLUSIVE MODE;

DO $$
DECLARE
    dup text;
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint
               WHERE conrelid = 'db_sys_config'::regclass AND conname = 'db_sys_config_sort_key_key') THEN
        RAISE EXCEPTION '参数表已经有唯一约束 db_sys_config_sort_key_key：这个库已经升级过。升级中止，什么都没有改';
    END IF;
    SELECT string_agg(format('%s.%s（%s 行）', sort, key, n), '，' ORDER BY sort, key) INTO dup
    FROM (SELECT sort, key, count(*) AS n FROM db_sys_config GROUP BY sort, key HAVING count(*) > 1) t;
    IF dup IS NOT NULL THEN
        RAISE EXCEPTION '参数表里有重复的参数：%。先删掉多余的行，再执行。升级中止，什么都没有改', dup;
    END IF;
END $$;

ALTER TABLE db_sys_config ADD CONSTRAINT db_sys_config_sort_key_key UNIQUE (sort, key);
DROP INDEX IF EXISTS idx_db_sys_config__sort_key;

COMMIT;
