-- 超级管理角色 r_AdminSuper 补上在线用户、操作日志两页的权限：
--   m_systokenlist、m_systokenlist:list、m_systokenlist:del、
--   m_syslogopt、m_syslogopt:list、m_syslogopt:del、m_syslogopt:clear
-- 补完之后这个角色的权限和 0002_seed_data.sql 里的是同一组（先后不同：脚本把缺的接在最后）。
-- 原来的 0002 漏了这 7 个：除了 user_id 是 superadmin 的默认管理员（这个账号不查权限），别的超级管理账号打不开这两页。
-- 给 r_AdminSuper 还缺这几个权限的库执行一次。拿不准就执行：都有了会停在开头的检查上，什么都不改。新建的库直接用 0001、0002，不用执行它。
-- 和别的升级脚本没有先后关系。
--
-- 执行之前：不用停服务，库和程序谁先换都行。
--   已经登录着的账号，重新登录之后才有这两页：账号的权限在登录时取（在角色管理页保存权限也是这样）。
--
-- 做的事在一个事务里：这 7 个里角色还没有的，按上面的顺序接在它的权限后面；已经有的不重复加，别的权限原样、顺序不变。
-- 角色表里没有 r_AdminSuper、或者 7 个都已经有了，停在开头的检查上，什么都不改。
SET client_encoding TO 'UTF8';

BEGIN;

DO $$
DECLARE
    keys text[] := ARRAY['m_systokenlist', 'm_systokenlist:list', 'm_systokenlist:del',
                         'm_syslogopt', 'm_syslogopt:list', 'm_syslogopt:del', 'm_syslogopt:clear'];
    cur  text[];
BEGIN
    SELECT menuset INTO cur FROM db_sys_role WHERE code = 'r_AdminSuper' FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION '角色表里没有 r_AdminSuper。升级中止，什么都没有改';
    END IF;
    IF keys <@ cur THEN
        RAISE EXCEPTION 'r_AdminSuper 已经有这 7 个权限：这个库已经升级过。升级中止，什么都没有改';
    END IF;
    UPDATE db_sys_role
    SET menuset = cur || ARRAY(SELECT k FROM unnest(keys) WITH ORDINALITY AS t(k, n) WHERE k <> ALL (cur) ORDER BY n)
    WHERE code = 'r_AdminSuper';
END $$;

COMMIT;
