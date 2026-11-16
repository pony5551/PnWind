import { http } from "@/utils/http";
import { sysrole_url } from "@/api/sysmgr/sysmgr_define";
const pageUrl = sysrole_url;

type Result = {
  code: number;
  message: string;
  data?: Array<any>;
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

/** 角色列表（全量，返回平铺数组，前端构建树） */
export const actRoleList = (data?: object) => {
  const lurl = pageUrl.concat("/list");
  return http.request<Result>("post", lurl, {
    data
  });
};

/** 获取角色菜单 */
export const actMenuSetGet = (data?: object) => {
  const lurl = pageUrl.concat("/menusetget");
  return http.request<ResultTable>("post", lurl, {
    data
  });
};

/** 保存角色菜单 */
export const actMenuSetSave = (data?: object) => {
  const lurl = pageUrl.concat("/menusetsave");
  return http.request<Result>("post", lurl, {
    data
  });
};

/** 新增 */
export const actRoleAdd = (data?: object) => {
  const lurl = pageUrl.concat("/add");
  return http.request<Result>("post", lurl, {
    data
  });
};

/** 编辑 */
export const actRoleEdit = (data?: object) => {
  const lurl = pageUrl.concat("/edit");
  return http.request<Result>("post", lurl, {
    data
  });
};

/** 删除 */
export const actRoleDel = (data?: object) => {
  const lurl = pageUrl.concat("/del");
  return http.request<Result>("post", lurl, {
    data
  });
};
