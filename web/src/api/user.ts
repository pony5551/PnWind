import { http } from "@/utils/http";
import { userauth_url } from "@/api/sysmgr/sysmgr_define";
import { getToken, formatToken } from "@/utils/auth";
const userauthUrl = userauth_url;

export type UserResult = {
  code: number;
  message: string;
  data: {
    /** 头像 */
    avatar: string;
    /** 用户名 */
    username: string;
    /** 昵称 */
    nickname: string;
    /** 当前登录用户的角色 */
    roles: Array<string>;
    /** 按钮级别权限 */
    permissions: Array<string>;
    /** `token` */
    accessToken: string;
    /** 用于调用刷新`accessToken`的接口时所需的`token` */
    refreshToken: string;
    /** `accessToken`的过期时间（格式'xxxx/xx/xx xx:xx:xx'） */
    expires: Date;
    tokenid: string;
    MenuSets: Array<any>;
  };
};

export type RefreshTokenResult = {
  code: number;
  message: string;
  data: {
    /** `token` */
    accessToken: string;
    /** 用于调用刷新`accessToken`的接口时所需的`token` */
    refreshToken: string;
    /** `accessToken`的过期时间（格式'xxxx/xx/xx xx:xx:xx'） */
    expires: Date;
  };
};

export type UserInfo = {
  /** 头像 */
  avatar: string;
  /** 用户名 */
  username: string;
  /** 昵称 */
  nickname: string;
  /** 邮箱 */
  email: string;
  /** 联系电话 */
  phone: string;
  /** 简介 */
  description: string;
};

export type UserInfoResult = {
  code: number;
  message: string;
  data: UserInfo;
};

type ResultTable = {
  code: number;
  message: string;
  data?: {
    /** 列表数据 */
    list: Array<any>;
    /** 总条目数 */
    total?: number;
    /** 每页显示条目个数 */
    pageSize?: number;
    /** 当前页数 */
    currentPage?: number;
  };
};

/** 登录 */
export const getLogin = (data?: object) => {
  const lurl = userauthUrl.concat("/login");
  return http.request<UserResult>("post", lurl, {
    data
  });
};

/** 刷新`token` */
export const refreshTokenApi = (data?: object) => {
  return http.request<RefreshTokenResult>("post", "/refresh-token", { data });
};

/** 退出登录（后端删除在线Token记录，使令牌立即失效）
 *  通过 beforeRequestCallback 绕过默认请求拦截器：
 *  避免 token 已过期时触发"刷新失败→logOut→再登出"的循环，且过期令牌也能完成后端清理 */
export const getLogout = () => {
  const lurl = userauthUrl.concat("/logout");
  const data = getToken();
  return http.request<{ code: number; message: string }>(
    "post",
    lurl,
    {},
    {
      beforeRequestCallback: config => {
        if (data?.accessToken) {
          config.headers["Authorization"] = formatToken(data.accessToken);
        }
      }
    }
  );
};

/** 账户设置-个人信息 */
export const getMine = (data?: object) => {
  return http.request<UserInfoResult>("get", "/mine", { data });
};

/** 账户设置-个人安全日志 */
export const getMineLogs = (data?: object) => {
  return http.request<ResultTable>("get", "/mine-logs", { data });
};
