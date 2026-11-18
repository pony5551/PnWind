-- PnWind 的系统表：管理员、角色、菜单、部门、参数、令牌、操作日志、用户，另有一张分区表演示。
-- 在空库上执行本文件建表，再执行 0002_seed_data.sql 写入初始数据。
-- 参数表的序列没有挂在列上，单独建立。

-- 管理员表。password2 是二次密码：一个账号只能有一处登录，已经在别处登录着再登录要输它。
CREATE SEQUENCE db_sys_admin_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_admin (
    id            bigint       NOT NULL DEFAULT nextval('db_sys_admin_id_seq'),
    user_id       varchar(50)  NOT NULL,               -- 管理员的用户 id，令牌表、操作日志表里记的就是它
    username      varchar(100) NOT NULL,               -- 登录名
    password      varchar(50)  NOT NULL,
    password2     varchar(50)  NOT NULL DEFAULT '',    -- 二次密码；空串表示没有单独设置，等同登录密码
    real_name     varchar(100) NOT NULL DEFAULT '',
    gender        smallint     NOT NULL DEFAULT -1,
    avatar        text         NOT NULL DEFAULT '',
    phone         varchar(50)  NOT NULL DEFAULT '',
    email         varchar(100) NOT NULL DEFAULT '',
    dept_id       bigint       NOT NULL DEFAULT 0,
    roles         text[]       NOT NULL DEFAULT '{}',  -- 角色编码
    status        integer      NOT NULL DEFAULT 0,     -- 1 启用，0 停用，-1 禁用，-2 冻结
    last_login_ip varchar(100) NOT NULL DEFAULT '',
    last_login_at timestamptz,
    created_ip    varchar(100) NOT NULL,
    created_at    timestamptz  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    timestamptz,                         -- NULL 表示没有删除
    CONSTRAINT db_sys_admin_pkey PRIMARY KEY (id),
    CONSTRAINT db_sys_admin_user_id_key UNIQUE (user_id),
    CONSTRAINT db_sys_admin_username_key UNIQUE (username)
);
ALTER SEQUENCE db_sys_admin_id_seq OWNED BY db_sys_admin.id;
COMMENT ON TABLE db_sys_admin IS '管理员';
COMMENT ON COLUMN db_sys_admin.user_id IS '用户id';
COMMENT ON COLUMN db_sys_admin.username IS '登录名';
COMMENT ON COLUMN db_sys_admin.password IS '密码';
COMMENT ON COLUMN db_sys_admin.password2 IS '二次密码，空串表示等同登录密码';
COMMENT ON COLUMN db_sys_admin.real_name IS '真实姓名';
COMMENT ON COLUMN db_sys_admin.gender IS '性别';
COMMENT ON COLUMN db_sys_admin.avatar IS '头像';
COMMENT ON COLUMN db_sys_admin.phone IS '手机号';
COMMENT ON COLUMN db_sys_admin.email IS '邮箱';
COMMENT ON COLUMN db_sys_admin.dept_id IS '部门id';
COMMENT ON COLUMN db_sys_admin.roles IS '角色编码';
COMMENT ON COLUMN db_sys_admin.status IS '状态：1 启用，0 停用，-1 禁用，-2 冻结';
COMMENT ON COLUMN db_sys_admin.last_login_ip IS '最近一次登录的ip';
COMMENT ON COLUMN db_sys_admin.last_login_at IS '最近一次登录的时间';
COMMENT ON COLUMN db_sys_admin.created_ip IS '创建时的ip';
COMMENT ON COLUMN db_sys_admin.created_at IS '创建时间';
COMMENT ON COLUMN db_sys_admin.deleted_at IS '删除时间，NULL 表示没有删除';

-- 角色表。
CREATE SEQUENCE db_sys_role_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_role (
    id         bigint      NOT NULL DEFAULT nextval('db_sys_role_id_seq'),
    parent_id  bigint      NOT NULL DEFAULT 0,       -- 上级角色的 id，0 表示顶级
    name       varchar(50) NOT NULL,
    code       varchar(50) NOT NULL,                 -- 角色编码，管理员表的 roles 里存的就是它
    sort       integer     NOT NULL DEFAULT 0,
    status     smallint    NOT NULL DEFAULT 0,       -- 1 启用，0 停用
    remark     text        NOT NULL DEFAULT '',
    menuset    text[]      NOT NULL DEFAULT '{}',    -- 权限键：菜单表里页面行、按钮行的 name
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT db_sys_role_pkey PRIMARY KEY (id),
    CONSTRAINT db_sys_role_code_key UNIQUE (code)
);
ALTER SEQUENCE db_sys_role_id_seq OWNED BY db_sys_role.id;
COMMENT ON TABLE db_sys_role IS '角色';
COMMENT ON COLUMN db_sys_role.parent_id IS '上级角色的id，0 表示顶级';
COMMENT ON COLUMN db_sys_role.name IS '角色名称';
COMMENT ON COLUMN db_sys_role.code IS '角色编码';
COMMENT ON COLUMN db_sys_role.sort IS '排序';
COMMENT ON COLUMN db_sys_role.status IS '状态：1 启用，0 停用';
COMMENT ON COLUMN db_sys_role.remark IS '备注';
COMMENT ON COLUMN db_sys_role.menuset IS '权限键：菜单表里页面行、按钮行的 name';
COMMENT ON COLUMN db_sys_role.created_at IS '创建时间';

CREATE SEQUENCE db_sys_menus_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_menus (
    id bigint NOT NULL DEFAULT nextval('db_sys_menus_id_seq'::regclass),
    menu_type smallint NOT NULL DEFAULT 0,
    parent_id bigint NOT NULL DEFAULT 0,
    title character varying(255) NOT NULL,
    name character varying(100) NOT NULL,
    path character varying(500),
    component character varying(500),
    rank integer NOT NULL DEFAULT 1,
    redirect character varying(500),
    icon character varying(100),
    extra_icon character varying(100),
    enter_transition character varying(100),
    leave_transition character varying(100),
    active_path character varying(500),
    auths jsonb,
    frame_src character varying(1000),
    frame_loading boolean DEFAULT true,
    keep_alive boolean DEFAULT false,
    hidden_tag boolean DEFAULT false,
    fixed_tag boolean DEFAULT false,
    show_link boolean DEFAULT true,
    show_parent boolean DEFAULT false,
    loged boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT db_sys_menus_pkey PRIMARY KEY (id)
);
ALTER SEQUENCE db_sys_menus_id_seq OWNED BY db_sys_menus.id;
CREATE UNIQUE INDEX idx_db_sys_menus_name ON db_sys_menus USING btree (name);
CREATE INDEX idx_db_sys_menus_parent_id ON db_sys_menus USING btree (parent_id);
CREATE INDEX idx_db_sys_menus_path ON db_sys_menus USING btree (path);
CREATE INDEX idx_db_sys_menus_rank ON db_sys_menus USING btree (rank);
COMMENT ON TABLE db_sys_menus IS '系统菜单路由配置表';
COMMENT ON COLUMN db_sys_menus.id IS '主键 ID';
COMMENT ON COLUMN db_sys_menus.menu_type IS '菜单类型：0=菜单，1=iframe，2=外链，3=按钮';
COMMENT ON COLUMN db_sys_menus.parent_id IS '父级菜单 ID，0 为根节点';
COMMENT ON COLUMN db_sys_menus.title IS '菜单名称（兼容国际化，若为 key 需在 locales 中配置）';
COMMENT ON COLUMN db_sys_menus.name IS '路由名称（必须唯一，需与组件 defineOptions name 一致）';
COMMENT ON COLUMN db_sys_menus.path IS '路由路径';
COMMENT ON COLUMN db_sys_menus.component IS '组件路径（不传时默认同 path）';
COMMENT ON COLUMN db_sys_menus.rank IS '菜单排序（home 路由为 0，其他建议从 1 开始）';
COMMENT ON COLUMN db_sys_menus.redirect IS '路由重定向地址';
COMMENT ON COLUMN db_sys_menus.icon IS '菜单图标';
COMMENT ON COLUMN db_sys_menus.extra_icon IS '右侧额外图标';
COMMENT ON COLUMN db_sys_menus.enter_transition IS '进场动画（页面加载动画）';
COMMENT ON COLUMN db_sys_menus.leave_transition IS '离场动画（页面加载动画）';
COMMENT ON COLUMN db_sys_menus.active_path IS '指定激活菜单的 path（解决传参路由高亮问题）';
COMMENT ON COLUMN db_sys_menus.auths IS '权限标识集合（按钮级别权限，JSON 数组格式）';
COMMENT ON COLUMN db_sys_menus.frame_src IS 'iframe 内嵌链接地址';
COMMENT ON COLUMN db_sys_menus.frame_loading IS '内嵌 iframe 是否开启首次加载动画';
COMMENT ON COLUMN db_sys_menus.keep_alive IS '是否缓存该路由页面';
COMMENT ON COLUMN db_sys_menus.hidden_tag IS '是否禁止添加到标签页';
COMMENT ON COLUMN db_sys_menus.fixed_tag IS '是否固定显示在标签页且不可关闭';
COMMENT ON COLUMN db_sys_menus.show_link IS '是否显示该菜单';
COMMENT ON COLUMN db_sys_menus.show_parent IS '是否显示父级菜单（针对只有一个子菜单的父级）';
COMMENT ON COLUMN db_sys_menus.loged IS '记录日志';

-- 部门表。
CREATE SEQUENCE db_sys_dept_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_dept (
    id bigint NOT NULL DEFAULT nextval('db_sys_dept_id_seq'::regclass),
    parent_id bigint NOT NULL DEFAULT 0,
    name character varying(50) NOT NULL,
    principal character varying(50),
    phone character varying(50),
    email character varying(100),
    sort integer NOT NULL DEFAULT 0,
    status smallint NOT NULL DEFAULT 0,
    remark text,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT db_sys_dept_pkey PRIMARY KEY (id),
    CONSTRAINT db_sys_dept_name_key UNIQUE (name)
);
ALTER SEQUENCE db_sys_dept_id_seq OWNED BY db_sys_dept.id;
COMMENT ON COLUMN db_sys_dept.parent_id IS '上级部门';
COMMENT ON COLUMN db_sys_dept.name IS '部门名称';
COMMENT ON COLUMN db_sys_dept.principal IS '部门负责人';
COMMENT ON COLUMN db_sys_dept.phone IS '手机号';
COMMENT ON COLUMN db_sys_dept.email IS '邮箱';
COMMENT ON COLUMN db_sys_dept.sort IS '排序';
COMMENT ON COLUMN db_sys_dept.status IS '状态';
COMMENT ON COLUMN db_sys_dept.remark IS '备注';

-- 参数表。唯一约束 (sort, key)：分组加键不能重复，程序按"分组.键"取参数的值。
CREATE SEQUENCE db_sys_config_config_id_seq START WITH 100000 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_config (
    config_id bigint NOT NULL DEFAULT nextval('db_sys_config_config_id_seq'::regclass),
    sort character varying(50) NOT NULL,
    key character varying(50) NOT NULL,
    value character varying(4000),
    remark character varying(255),
    patrn character varying(255),
    expansion character varying(255),
    CONSTRAINT db_sys_config_pkey PRIMARY KEY (config_id),
    CONSTRAINT db_sys_config_sort_key_key UNIQUE (sort, key)
);

-- 令牌表：登录一次一行，鉴权时每个请求按 token_id 找到它。ip 只记地址、不带端口。
-- 唯一约束 (user_type, user_id)：一个账号最多一行，也就是只能有一处登录。
CREATE SEQUENCE db_sys_tokenlist_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_tokenlist (
    id          bigint       NOT NULL DEFAULT nextval('db_sys_tokenlist_id_seq'),
    token_id    varchar(50)  NOT NULL,                -- 令牌编号，签在令牌里；每个请求按它找这一行
    user_type   smallint     NOT NULL,                -- 1 管理员，2 用户
    user_id     varchar(50)  NOT NULL,                -- 管理员表或用户表的 user_id
    roles       text[]       NOT NULL DEFAULT '{}',   -- 登录时的角色编码
    menuset     text[]       NOT NULL DEFAULT '{}',   -- 登录时按角色合并出来的权限键
    ip          varchar(100) NOT NULL DEFAULT '',     -- 最近一次请求的地址，不带端口
    created_at  timestamptz  NOT NULL DEFAULT CURRENT_TIMESTAMP,   -- 登录时间
    updated_at  timestamptz  NOT NULL DEFAULT CURRENT_TIMESTAMP,   -- 最近一次请求的时间
    CONSTRAINT db_sys_tokenlist_pkey PRIMARY KEY (id),
    CONSTRAINT db_sys_tokenlist_token_id_key UNIQUE (token_id),
    CONSTRAINT db_sys_tokenlist_user_type_user_id_key UNIQUE (user_type, user_id)  -- 一个账号最多一行
);
ALTER SEQUENCE db_sys_tokenlist_id_seq OWNED BY db_sys_tokenlist.id;
COMMENT ON TABLE db_sys_tokenlist IS '登录令牌';
COMMENT ON COLUMN db_sys_tokenlist.token_id IS '令牌编号';
COMMENT ON COLUMN db_sys_tokenlist.user_type IS '用户类型：1 管理员，2 用户';
COMMENT ON COLUMN db_sys_tokenlist.user_id IS '用户id';
COMMENT ON COLUMN db_sys_tokenlist.roles IS '登录时的角色编码';
COMMENT ON COLUMN db_sys_tokenlist.menuset IS '登录时按角色合并出来的权限键';
COMMENT ON COLUMN db_sys_tokenlist.ip IS '最近一次请求的ip';
COMMENT ON COLUMN db_sys_tokenlist.created_at IS '登录时间';
COMMENT ON COLUMN db_sys_tokenlist.updated_at IS '最近一次请求的时间';

-- 操作日志表。
CREATE SEQUENCE db_sys_logopt_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_logopt (
    id          bigint       NOT NULL DEFAULT nextval('db_sys_logopt_id_seq'),
    device_id   bigint       NOT NULL DEFAULT 0,      -- 节点号，取自配置里的 DeviceId
    user_type   smallint     NOT NULL,                -- 1 管理员，2 用户
    user_id     varchar(50)  NOT NULL,                -- 管理员表或用户表的 user_id
    type        varchar(50)  NOT NULL,                -- 类别，现在只有"用户操作"
    events      varchar(50)  NOT NULL,                -- 页面事件，请求地址的最后一段：list、add、edit……
    details     text         NOT NULL DEFAULT '',     -- 页面事件详情：菜单标题-按钮标题，例如 部门管理-添加
    page_path   text         NOT NULL,                -- 页面的接口地址，例如 /web/sysmgr/sysset/sysdept
    page_query  text         NOT NULL DEFAULT '',     -- 地址参数
    page_form   text         NOT NULL DEFAULT '',     -- 请求体（最多记 64KB）
    ip          varchar(100) NOT NULL DEFAULT '',
    created_at  timestamptz  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT db_sys_logopt_pkey PRIMARY KEY (id)
);
ALTER SEQUENCE db_sys_logopt_id_seq OWNED BY db_sys_logopt.id;
CREATE INDEX idx_db_sys_logopt__created_at ON db_sys_logopt USING btree (created_at);
COMMENT ON TABLE db_sys_logopt IS '操作日志';
COMMENT ON COLUMN db_sys_logopt.device_id IS '节点号';
COMMENT ON COLUMN db_sys_logopt.user_type IS '用户类型：1 管理员，2 用户';
COMMENT ON COLUMN db_sys_logopt.user_id IS '用户id';
COMMENT ON COLUMN db_sys_logopt.type IS '类别，例如 用户操作';
COMMENT ON COLUMN db_sys_logopt.events IS '页面事件，例如 list、add、edit';
COMMENT ON COLUMN db_sys_logopt.details IS '页面事件详情，例如 部门管理-添加';
COMMENT ON COLUMN db_sys_logopt.page_path IS '页面的接口地址';
COMMENT ON COLUMN db_sys_logopt.page_query IS '地址参数';
COMMENT ON COLUMN db_sys_logopt.page_form IS '请求体';
COMMENT ON COLUMN db_sys_logopt.ip IS 'ip';
COMMENT ON COLUMN db_sys_logopt.created_at IS '操作时间';

-- 用户表。
CREATE SEQUENCE db_sys_users_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE SEQUENCE db_sys_users_user_id_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;
CREATE TABLE db_sys_users (
    id             bigint        NOT NULL DEFAULT nextval('db_sys_users_id_seq'),
    user_id        varchar(50)   NOT NULL DEFAULT nextval('db_sys_users_user_id_seq')::varchar,  -- 用户编号，新增时由序列给
    name           varchar(128)  NOT NULL,                -- 名称：个人填姓名，企业填全称；不是登录名
    type           smallint      NOT NULL DEFAULT 1,      -- 1 个人，2 个体户，3 企业
    email          varchar(128)  NOT NULL,
    phone          varchar(20)   NOT NULL,
    password       varchar(255)  NOT NULL DEFAULT '',
    pay_password   varchar(255)  NOT NULL DEFAULT '',     -- 资金密码
    roles          text[]        NOT NULL DEFAULT '{}',   -- 角色编码
    status         smallint      NOT NULL DEFAULT 0,      -- 0 待审核，1 正常，2 冻结，3 注销
    verified_id    bigint        NOT NULL DEFAULT 0,      -- 实名信息的 id，0 表示还没有实名（实名信息表以后设计，实名后绑定）
    cash_balance   numeric(20,6) NOT NULL DEFAULT 0,      -- 现金余额
    funds_balance  numeric(20,6) NOT NULL DEFAULT 0,      -- 资金余额（可提现）
    frozen_balance numeric(20,6) NOT NULL DEFAULT 0,      -- 冻结金额
    config         jsonb         NOT NULL DEFAULT '{}',   -- 扩展配置
    remark         text          NOT NULL DEFAULT '',
    last_login_ip  varchar(100)  NOT NULL DEFAULT '',
    last_login_at  timestamptz,
    created_ip     varchar(100)  NOT NULL DEFAULT '',
    created_at     timestamptz   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     timestamptz   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at     timestamptz,                           -- NULL 表示没有删除
    CONSTRAINT db_sys_users_pkey PRIMARY KEY (id),
    CONSTRAINT db_sys_users_user_id_key UNIQUE (user_id),
    CONSTRAINT db_sys_users_email_key UNIQUE (email),
    CONSTRAINT db_sys_users_phone_key UNIQUE (phone)
);
ALTER SEQUENCE db_sys_users_id_seq OWNED BY db_sys_users.id;
ALTER SEQUENCE db_sys_users_user_id_seq OWNED BY db_sys_users.user_id;
COMMENT ON TABLE db_sys_users IS '用户';
COMMENT ON COLUMN db_sys_users.user_id IS '用户编号';
COMMENT ON COLUMN db_sys_users.name IS '名称：个人填姓名，企业填全称';
COMMENT ON COLUMN db_sys_users.type IS '用户类型：1 个人，2 个体户，3 企业';
COMMENT ON COLUMN db_sys_users.email IS '邮箱';
COMMENT ON COLUMN db_sys_users.phone IS '手机号';
COMMENT ON COLUMN db_sys_users.password IS '登录密码';
COMMENT ON COLUMN db_sys_users.pay_password IS '资金密码';
COMMENT ON COLUMN db_sys_users.roles IS '角色编码';
COMMENT ON COLUMN db_sys_users.status IS '状态：0 待审核，1 正常，2 冻结，3 注销';
COMMENT ON COLUMN db_sys_users.verified_id IS '实名信息id，0 表示还没有实名';
COMMENT ON COLUMN db_sys_users.cash_balance IS '现金余额';
COMMENT ON COLUMN db_sys_users.funds_balance IS '资金余额（可提现）';
COMMENT ON COLUMN db_sys_users.frozen_balance IS '冻结金额';
COMMENT ON COLUMN db_sys_users.config IS '扩展配置';
COMMENT ON COLUMN db_sys_users.remark IS '备注';
COMMENT ON COLUMN db_sys_users.last_login_ip IS '最近一次登录的ip';
COMMENT ON COLUMN db_sys_users.last_login_at IS '最近一次登录的时间';
COMMENT ON COLUMN db_sys_users.created_ip IS '创建时的ip';
COMMENT ON COLUMN db_sys_users.created_at IS '创建时间';
COMMENT ON COLUMN db_sys_users.updated_at IS '修改时间';
COMMENT ON COLUMN db_sys_users.deleted_at IS '删除时间，NULL 表示没有删除';

-- 分区表演示：一张按月分区的表，给分区管理器（core/pnpartmgr）做演示用。
-- 分区表没有自增的 id：第一列是单号（雪花 ID，由程序给），created_at 记创建的时刻，
-- 另有一列日期 created_date 专门当分区键，主键是 (order_no, created_date)。
-- 这里只建主表。每个月的分区（名字像 db_demo_partition_202611）由程序建：启动时一次，之后每天 3 点一次，
-- 范围是过去 1 个月到未来 3 个月（见 main.go）。
CREATE TABLE db_demo_partition (
    order_no      bigint       NOT NULL,
    created_at    timestamptz  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_date  date         NOT NULL DEFAULT CURRENT_DATE,
    order_title   varchar(200) NOT NULL DEFAULT '',
    CONSTRAINT db_demo_partition_pkey PRIMARY KEY (order_no, created_date)
) PARTITION BY RANGE (created_date);
COMMENT ON TABLE db_demo_partition IS '分区表演示（按 created_date 按月分区）';
COMMENT ON COLUMN db_demo_partition.order_no IS '单号（雪花ID），主键第一列';
COMMENT ON COLUMN db_demo_partition.created_at IS '记录创建时刻';
COMMENT ON COLUMN db_demo_partition.created_date IS '分区键、主键第二列（创建日期）';
COMMENT ON COLUMN db_demo_partition.order_title IS '标题';
